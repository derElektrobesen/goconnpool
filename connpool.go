// Package goconnpool implements connections pool with ratelimits and backoff for broken connections.
//
// Connection returned by the pool is protocol-independent.
//
package goconnpool

import (
	"context"
	"net"
)

// Conn is a wrapper around net.Conn interface
type Conn interface {
	net.Conn

	// ReturnToPool returns connection back into pool.
	// This method should be called to reuse already opened connection.
	//
	// To prevent connections leaking eigther Close() or ReturnToPool() methods shuld be called.
	//
	// Connection shouldn't be used after returning to pool.
	ReturnToPool() error

	// OriginalConn returns original connection returned by the dialer.
	// Could be useful when the connection have the specific type and only this type could be used to interact with
	// server.
	//
	// XXX: Returned connection shouldn't be closed.
	OriginalConn() net.Conn
}

// ConnPool is the base interface to interact with user.
type ConnPool interface {
	// OpenConnNonBlock requests one connection from the pool.
	//
	// If the pool already contains opened connection, this connection will be returned.
	//
	// If each registered server is down, function returns an error.
	// Otherwise function returns active connection (to some alive server in round-robin order)
	// which could be used to send any type of request.
	//
	// Connection should be closed (with Close() call) or returned into pool (with ReturnToPool() call) after use.
	//
	// Pool regulates number of requests per server using MaxRPS config variable.
	// To prevent breaking this mechanism down, don't try to send multiple number of requests
	// into one connection: close previous connection and take one more connection again.
	//
	// Returned error couldn't contain anough information about any server status and therefore
	// Logger was used. Don't forget to setup Logger if you want to know this info.
	OpenConnNonBlock(ctx context.Context) (Conn, error)

	// OpenConn does same things as OpenConnNonBlock, but it blocks until new connection
	// will be established. This process could be cancelled using the context.
	OpenConn(ctx context.Context) (Conn, error)

	// RegisterServer registers new server in connections pool.
	// This server stands into round-robin queue to be used during OpenConn call.
	//
	// This operation is a part of initialization.
	// Don't try to call it in runtime: not thread safe.
	RegisterServer(addr string)
}

// NewConnPool creates new pool with configuration passed.
func NewConnPool(cfg Config) ConnPool {
	return newConnPool(cfg)
}
