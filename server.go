package goconnpool

import (
	"context"
	"net"
	"sync"
	"time"

	"github.com/cenkalti/backoff"
	"github.com/pkg/errors"
)

type dialer interface {
	DialContext(ctx context.Context, network, address string) (net.Conn, error)
}

type connectionProvider interface {
	getConnection(ctx context.Context) (Conn, error)
}

type server struct {
	mu sync.Mutex

	network string
	addr    string

	nOpenedConns int
	maxConns     int
	openedConns  deck

	reqDuration time.Duration
	lastUsage   time.Time

	dialer dialer

	bOff        backoff.BackOff
	nextBackoff time.Time
	down        bool

	clock Clock
}

func newServerWrapper(network, addr string, cfg Config, dialer dialer) connectionProvider {
	return newServer(network, addr, cfg, dialer)
}

func newServer(network, addr string, cfg Config, dialer dialer) *server {
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
		network:  network,
		addr:     addr,
		maxConns: cfg.MaxConnsPerServer,
		dialer:   dialer,
		bOff:     bc,

		reqDuration: time.Duration(1000000.0/float64(cfg.MaxRPS)) * time.Microsecond,

		clock: cfg.Clock,
	}
}

func (s *server) updateLastUsage() bool {
	if s.clock.Since(s.lastUsage) > s.reqDuration {
		s.lastUsage = s.clock.Now()
		return true
	}

	return false
}

func (s *server) getConnection(ctx context.Context) (Conn, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.updateLastUsage() {
		return nil, errors.Wrap(errRatelimited, "too frequent request")
	}

	if s.openedConns.size() > 0 {
		return s.wrapServerConn(s.openedConns.pop().(net.Conn)), nil
	}

	if s.nOpenedConns >= s.maxConns {
		return nil, errors.Wrap(errRatelimited, "too many opened connections")
	}

	if s.down {
		waitFor := s.clock.Since(s.nextBackoff)
		if waitFor < 0 {
			waitFor *= time.Duration(-1)
			return nil, errors.Wrapf(errServerIsDown, "retry after %s", waitFor)
		}
	}

	// Trying to establish connection
	cn, err := s.dialer.DialContext(ctx, s.network, s.addr)
	if err != nil {
		s.down = true
		s.nextBackoff = s.clock.Now().Add(s.bOff.NextBackOff())
		return nil, errors.Wrapf(errServerIsDown,
			"can't establish %s connection to %s: %s", s.network, s.addr, err)
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
	s        *server
	unusable bool
}

func (s *server) wrapServerConn(cn net.Conn) Conn {
	return &serverConn{
		Conn: cn,
		s:    s,
	}
}

func (cn *serverConn) Close() error {
	cn.s.mu.Lock()
	defer cn.s.mu.Unlock()

	if cn.unusable {
		cn.s.nOpenedConns--
		return errors.WithStack(cn.Conn.Close())
	}

	cn.s.openedConns.push(cn.Conn)
	return nil
}

func (cn *serverConn) MarkBroken() {
	cn.unusable = true
}
