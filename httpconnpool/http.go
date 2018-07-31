// Package httpconnpool implements HTTP connections pool.
//
// This implementation is just a wrapper around goconnpool and could be used as a sample
// pool usage.
//
package httpconnpool

import (
	"bufio"
	"context"
	"net/http"
	"time"

	"github.com/derElektrobesen/goconnpool"
	"github.com/pkg/errors"
)

type roundTripper struct {
	p   goconnpool.ConnPool
	cfg Config
}

// HTTPConnPool describes HTTP connections pool. This interface could be embeded into http.Client structure.
type HTTPConnPool interface {
	http.RoundTripper

	// RegisterServer registers new server in connections pool.
	// This server stands into round-robin queue to be used during OpenConn call.
	//
	// This operation is a part of initialization.
	// Don't try to call it in runtime: not thread safe.
	RegisterServer(addr string)
}

// NewTransport returns a RoundTripper object to be used under http client.
// You should call RegisterServer() to setup one or more servers into pool.
func NewTransport(cfg Config) HTTPConnPool {
	cfg = cfg.WithDefaults()
	return &roundTripper{
		p:   goconnpool.NewConnPool(*cfg.PoolConfig),
		cfg: cfg,
	}
}

func (rt *roundTripper) connect(ctx context.Context) (goconnpool.Conn, error) {
	if rt.cfg.GetConnectionTimeout != Forever {
		var cancel func()
		ctx, cancel = context.WithTimeout(ctx, rt.cfg.GetConnectionTimeout)
		defer cancel()
	}

	cn, err := rt.p.OpenConn(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "can't open connection to send HTTP request")
	}

	return cn, nil
}

func (rt *roundTripper) sendRequest(cn goconnpool.Conn, req *http.Request) (*http.Response, error) {
	var deadline time.Time
	if rt.cfg.RequestTimeout != Forever {
		deadline = rt.cfg.PoolConfig.Clock.Now().Add(rt.cfg.RequestTimeout)
	}

	if err := cn.SetDeadline(deadline); err != nil {
		return nil, errors.Wrap(err, "can't set request deadline")
	}

	if err := req.Write(cn); err != nil {
		return nil, errors.Wrap(err, "can't write request")
	}

	response, err := http.ReadResponse(bufio.NewReader(cn), req)
	return response, errors.Wrap(err, "can't read response")
}

func (rt *roundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	cn, err := rt.connect(context.Background())
	if err != nil {
		return nil, errors.Wrap(err, "can't connect")
	}

	response, err := rt.sendRequest(cn, req)
	if err != nil {
		if closeErr := cn.Close(); closeErr != nil {
			rt.cfg.PoolConfig.Logger.Errorf("can't close connection: %s", err)
		}

		return nil, errors.Wrap(err, "can't send request")
	}

	cn.ReturnToPool()

	return response, nil
}

func (rt *roundTripper) RegisterServer(addr string) {
	rt.p.RegisterServer(addr)
}
