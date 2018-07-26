package goconnpool

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/pkg/errors"
)

type connPool struct {
	cfg Config

	servers             roundRobin
	connProviderFactory func(network string, addr string, cfg Config, dialer dialer) connectionProvider
}

func newConnPool(cfg Config) *connPool {
	cfg = cfg.withDefaults()

	return &connPool{
		cfg:                 cfg,
		connProviderFactory: newServerWrapper, // required for tests
	}
}

func (p *connPool) OpenConn(ctx context.Context) (Conn, error) {
	for {
		cn, timeout, err := p.openConn(ctx)
		if err == nil {
			return cn, nil
		}

		p.cfg.Logger.Printf("can't connect to servers: %s; retry after %s", err, timeout)

		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("operation cancelled")
		case <-time.After(timeout):
		}
	}
}

func (p *connPool) OpenConnNonBlock(ctx context.Context) (Conn, error) {
	cn, _, err := p.openConn(ctx)
	return cn, err
}

func (p *connPool) openConn(ctx context.Context) (Conn, time.Duration, error) {
	var (
		hasDown        bool
		hasRatelimited bool
		globErr        error
		maxTimeout     time.Duration
	)

	for i := 0; i < p.servers.size(); i++ {
		s := p.servers.next().(connectionProvider)

		cn, err := s.getConnection(ctx)
		if err == nil {
			return cn, 0, nil
		}

		switch errors.Cause(err) {
		case errServerIsDown:
			p.cfg.Logger.Printf("can't connect to server: %s", err)
			hasDown = true
		case errRatelimited:
			hasRatelimited = true
		default:
			p.cfg.Logger.Printf("can't connect to server: %s", err)
			globErr = err
		}

		waitFor := s.retryTimeout()
		if maxTimeout == 0 || maxTimeout > waitFor {
			maxTimeout = waitFor
		}
	}

	if hasDown && hasRatelimited {
		globErr = errors.New("some servers are down, other ratelimited")
	} else if hasDown {
		globErr = errors.New("all servers are down")
	} else if hasRatelimited {
		globErr = errors.New("all servers are ratelimited")
	}

	return nil, maxTimeout, globErr
}

func (p *connPool) RegisterServer(network string, addr string) {
	p.servers.push(p.connProviderFactory(network, addr, p.cfg, &net.Dialer{
		Timeout: p.cfg.ConnectTimeout,
	}))
}
