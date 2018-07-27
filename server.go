package goconnpool

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/cenkalti/backoff"
	"github.com/pkg/errors"
)

type connectionProvider interface {
	getConnection(ctx context.Context) (Conn, error)
	retryTimeout() time.Duration
}

type server struct {
	mu sync.Mutex

	addr           string
	connectTimeout time.Duration

	nOpenedConns int
	maxConns     int
	openedConns  deck

	reqDuration time.Duration
	lastUsage   time.Time

	dialer Dialer

	bOff        backoff.BackOff
	nextBackoff time.Time
	down        bool

	clock Clock
}

var (
	errRatelimit    = fmt.Errorf("ratelimit")
	errServerIsDown = fmt.Errorf("server is down")
)

func newServerWrapper(addr string, cfg Config) connectionProvider {
	return newServer(addr, cfg)
}

func newServer(addr string, cfg Config) *server {
	bc := backoff.NewExponentialBackOff()
	bc.InitialInterval = cfg.InitialBackoffInterval
	bc.MaxInterval = cfg.MaxBackoffInterval
	bc.MaxElapsedTime = 0
	bc.Clock = cfg.Clock

	bc.Reset() // required to re-setup config options

	if cfg.backoffRandomizationFactor != nil {
		// only for tests. Default backoff interval should be used in production
		bc.RandomizationFactor = *cfg.backoffRandomizationFactor
	}

	return &server{
		addr:     addr,
		maxConns: cfg.MaxConnsPerServer,
		dialer:   cfg.Dialer,
		bOff:     bc,

		connectTimeout: cfg.ConnectTimeout,

		reqDuration: time.Duration(1000000.0/float64(cfg.MaxRPS)) * time.Microsecond,

		clock: cfg.Clock,
	}
}

func (s *server) updateLastUsage() bool {
	// XXX: Function should be called under mutex

	if s.getRatelimitTimeout() > 0 {
		return false
	}

	s.lastUsage = s.clock.Now()
	return true
}

func (s *server) retryTimeout() time.Duration {
	s.mu.Lock()
	defer s.mu.Unlock()

	var waitFor time.Duration
	if s.down {
		waitFor = s.getDownTimeout()
	}

	if waitFor == 0 {
		waitFor = s.getRatelimitTimeout()
	}

	if waitFor == 0 && s.nOpenedConns >= s.maxConns {
		// too many opened connections: can't open connection right now
		waitFor = 100 * time.Millisecond // TODO: move into config
	}

	return waitFor
}

func (s *server) getDownTimeout() time.Duration {
	waitFor := s.clock.Since(s.nextBackoff)
	if waitFor < 0 {
		waitFor *= time.Duration(-1)
		return waitFor
	}

	return 0
}

func (s *server) getRatelimitTimeout() time.Duration {
	waitFor := s.clock.Since(s.lastUsage) - s.reqDuration
	if waitFor < 0 {
		waitFor *= time.Duration(-1)
		return waitFor
	}

	return 0
}

func (s *server) makeConnection(ctx context.Context) (net.Conn, error) {
	var (
		cn  net.Conn
		err error
	)

	// Trying to establish connection
	ctx, cancel := context.WithTimeout(ctx, s.connectTimeout)
	defer cancel() // required to release context resources in case if ready chan was closed before timeout

	ready := make(chan struct{})
	go func() {
		cn, err = s.dialer.Dial(ctx, s.addr)
		close(ready)
	}()

	select {
	case <-ctx.Done():
		// Some corner-case when dial completes 'just in time' as ctx.Done()
		select {
		case <-ready:
			return cn, err
		default:
			return nil, errors.WithStack(fmt.Errorf("can't dial to %s: timeout", s.addr))
		}
	case <-ready:
		return cn, err
	}
}

func (s *server) getConnection(ctx context.Context) (Conn, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.updateLastUsage() {
		return nil, errors.Wrap(errRatelimit, "too frequent request")
	}

	if s.openedConns.size() > 0 {
		return s.wrapServerConn(s.openedConns.pop().(net.Conn)), nil
	}

	if s.nOpenedConns >= s.maxConns {
		return nil, errors.Wrap(errRatelimit, "too many opened connections")
	}

	if s.down {
		waitFor := s.getDownTimeout()
		if waitFor > 0 {
			// prevent too frequent connects here
			return nil, errors.Wrapf(errRatelimit, "retry after %s", waitFor)
		}
	}

	cn, err := s.makeConnection(ctx)
	if err != nil {
		s.down = true
		s.nextBackoff = s.clock.Now().Add(s.bOff.NextBackOff())
		return nil, errors.Wrapf(errServerIsDown,
			"can't establish connection to %s: %s", s.addr, err)
	}

	s.nOpenedConns++

	if s.down {
		s.down = false
		s.bOff.Reset()
	}

	return s.wrapServerConn(cn), nil
}

type serverConn struct {
	net.Conn
	s *server

	closed bool
	inPool bool
}

func (s *server) wrapServerConn(cn net.Conn) Conn {
	return &serverConn{
		Conn: cn,
		s:    s,
	}
}

func (cn *serverConn) checkCouldBeReturned() error {
	if cn.closed {
		return errors.New("connection already closed")
	}

	if cn.inPool {
		return errors.New("connection already returned in pool")
	}

	return nil
}

func (cn *serverConn) ReturnToPool() error {
	cn.s.mu.Lock()
	defer cn.s.mu.Unlock()

	if err := cn.checkCouldBeReturned(); err != nil {
		return errors.WithStack(err)
	}

	cn.inPool = true
	cn.s.openedConns.push(cn.Conn)

	return nil
}

func (cn *serverConn) Close() error {
	cn.s.mu.Lock()
	defer cn.s.mu.Unlock()

	if err := cn.checkCouldBeReturned(); err != nil {
		return errors.WithStack(err)
	}

	cn.s.nOpenedConns--
	cn.closed = true

	return errors.WithStack(cn.Conn.Close())
}

func (cn *serverConn) OriginalConn() net.Conn {
	return cn.Conn
}
