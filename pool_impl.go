package goconnpool

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/pkg/errors"
)

var (
	ErrNoServersRegistered = fmt.Errorf("no registered servers found")
)

type connPool struct {
	cfg Config

	mu sync.Mutex

	servers             roundRobin
	connProviderFactory func(addr string, cfg Config) connectionProvider
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

		if err == ErrNoServersRegistered {
			return nil, err
		}

		p.cfg.Logger.Printf("can't connect to servers: %s; retry after %s", err, timeout)

		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("operation cancelled")
		case <-p.cfg.Clock.After(timeout):
		}
	}
}

func (p *connPool) OpenConnNonBlock(ctx context.Context) (Conn, error) {
	cn, _, err := p.openConn(ctx)
	return cn, err
}

func (p *connPool) openConn(ctx context.Context) (Conn, time.Duration, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.servers.size() == 0 {
		return nil, 0, ErrNoServersRegistered
	}

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
		case errRatelimit:
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

func (p *connPool) RegisterServer(addr string) {
	p.servers.push(p.connProviderFactory(addr, p.cfg))
}
