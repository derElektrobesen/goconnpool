package goconnpool

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
)

type connPool struct {
	cfg Config

	servers roundRobin
}

func newConnPool(cfg Config) (*connPool, error) {
	cfg = cfg.withDefaults()

	return &connPool{
		cfg: cfg,
	}, nil
}

func (p *connPool) OpenConn(ctx context.Context) (Conn, error) {
	for i := 0; i < p.servers.size(); i++ {
		s := p.servers.next().(*server)

		cn, err := s.getConnection(ctx)
		if err == nil {
			return cn, nil
		}

		switch errors.Cause(err) {
		case errRatelimited:
			// do nothing
		case errServerIsDown:
			p.cfg.Logger.Printf("can't connect to server: %s", err)
		}
	}

	return nil, fmt.Errorf("all servers are down")
}

func (p *connPool) RegisterServer(network string, addr string) {
	p.servers.push(newServer(network, addr, p.cfg))
}
