package goconnpool

import (
	"context"
	"fmt"
	"net"
)

var (
	errRatelimited  = fmt.Errorf("ratelimited")
	errServerIsDown = fmt.Errorf("server is down")
)

// Conn is a wrapper around net.Conn interface
type Conn interface {
	net.Conn

	// MarkUnusable marks connection not usable any more:
	// connection will not be closed when Close() method will be invoked
	MarkUnusable()
}

// ConnPool is the base interface to interact with user.
type ConnPool interface {
	// GetConnection requests one connection from the pool.
	//
	// If the pool already contains opened connection, this connection will be returned.
	//
	// If each registered server is down, function returns an error.
	// Otherwise function returns active connection (to some alive server in round-robin order)
	// which could be used to send any type of request.
	//
	// Connection should be closed after usage. This action will return connection into pool.
	// If you understand the connection should be completely closed, call `conn.MarkUnusable` first.
	//
	// Pool regulates number of requests per server using `MaxRPS` config variable.
	// To prevent breaking this mechanism down, don't try to send multiple number of requests
	// into one connection: close previous connection and take one more connection again.
	//
	// Returned error couldn't contain anough information about any server status and therefore
	// Logger was used. Don't forget to setup Logger if you want to know this info.
	OpenConn(ctx context.Context) (Conn, error)

	// RegisterServer registers new server in connections pool.
	// This server stands into round-robin queue to be used during `OpenCall` conn.
	//
	// This operation is a part of initialization.
	// Don't try to call it in runtime: not thread safe.
	RegisterServer(network string, addr string)
}

// NewConnPool creates new pool with configuration passed.
func NewConnPool(cfg Config) (ConnPool, error) {
	return newConnPool(cfg)
}
