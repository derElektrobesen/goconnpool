package goconnpool

import (
	"context"
	"net"
)

// Dialer interface used to dial specific server independently from the specific protocol.
type Dialer interface {
	// Dial dials the address passed (this address is one of the registered servers addresses).
	//
	// Function should return net.Conn-compatible connection.
	//
	// Function should be implemented in the thread-safe way.
	// Connect timeout is setuped before this function invocation: be sure your function not stuck in the case of
	// timeout.
	//
	// address variable is the same address used into RegisterServer call.
	Dial(ctx context.Context, address string) (net.Conn, error)
}

// TCPDialer is the default implementation of Dialer interface.
// Use it for raw TCP connections.
type TCPDialer struct {
	d net.Dialer
}

// Dial dials some TCP server using net.Dialer structure.
func (d *TCPDialer) Dial(ctx context.Context, address string) (net.Conn, error) {
	return d.d.DialContext(ctx, "tcp", address)
}
