package goconnpool

import (
	"context"
	"net"
	"sync"
	"time"

	"github.com/cenkalti/backoff"
	"github.com/pkg/errors"
)

type server struct {
	mu sync.Mutex

	network string
	addr    string

	nOpenedConns int
	maxConns     int
	openedConns  deck

	reqDuration time.Duration
	lastUsage   time.Time

	dialer net.Dialer

	bOff        backoff.BackOff
	nextBackoff time.Time
	down        bool

	clock Clock
}

func newServer(network, addr string, cfg Config) *server {
	bc := backoff.NewExponentialBackOff()
	bc.InitialInterval = cfg.InitialBackoffInterval
	bc.MaxInterval = cfg.MaxBackoffInterval
	bc.MaxElapsedTime = 0

	return &server{
		network: network,
		addr:    addr,

		maxConns: cfg.MaxConnsPerServer,

		dialer: net.Dialer{
			Timeout: cfg.ConnectTimeout,
		},

		bOff: bc,

		reqDuration: time.Duration(1000.0/float64(cfg.MaxRPS)) * time.Millisecond,

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
		return s.openedConns.pop().(Conn), nil
	}

	if s.nOpenedConns > s.maxConns {
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

	s.openedConns.push(cn)
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
	err := cn.Conn.Close()

	cn.s.mu.Lock()
	defer cn.s.mu.Unlock()

	if err != nil || cn.unusable {
		cn.s.nOpenedConns--
		return errors.WithStack(err)
	}

	cn.s.openedConns.push(cn.Conn)
	return nil
}

func (cn *serverConn) MarkUnusable() {
	cn.unusable = true
}
