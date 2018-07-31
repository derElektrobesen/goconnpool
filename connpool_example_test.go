package goconnpool

import (
	"context"
	"flag"
	"fmt"
	net "net"
	"time"
)

func ExampleNewConnPool_base() {
	cfg := NewConfig(&flag.FlagSet{})
	flag.Parse() // XXX: This call is required to fill config variables

	// Create a pool
	pool := NewConnPool(*cfg)

	// Register some servers
	pool.RegisterServer("127.0.0.1:1234")
	pool.RegisterServer("8.8.8.8:1234")

	for i := 0; i < 10; i++ {
		cn, err := pool.OpenConnNonBlock(context.Background()) // Context could be cancelable here
		if err != nil {
			// All servers are down or ratelimited: try again later
			time.Sleep(100 * time.Millisecond)
			continue
		}

		if _, err := cn.Write([]byte("Hello")); err != nil {
			// Can't write the message to the server.
			// Force-close connection
			cn.Close()
			return
		}

		// This call moves a connection back to pool
		cn.ReturnToPool()
	}
}

func ExampleNewConnPool_blockingCalls() {
	cfg := NewConfig(&flag.FlagSet{})
	flag.Parse() // XXX: This call is required to fill config variables

	// Create a pool
	pool := NewConnPool(*cfg)

	// Register some servers
	pool.RegisterServer("127.0.0.1:1111")
	pool.RegisterServer("127.0.0.1:2222")

	// It is simplier to use WithTimeout() here ;)
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(10 * time.Second)
		cancel() // timeout
	}()

	// Following call will blocks until any connection (to port 1111 or to port 2222) will be established.
	// We will cancel the request if we still can't connect after 10 seconds
	cn, err := pool.OpenConn(ctx)
	if err != nil {
		// Timeout
	}

	defer cn.Close()

	// use your conn
}

type MyConn struct {
	net.Conn

	addr string
}

func (cn MyConn) Hello() string {
	return fmt.Sprintf("Hello from %s", cn.addr)
}

type MyDialer struct{}

func (MyDialer) Dial(ctx context.Context, addr string) (net.Conn, error) {
	return MyConn{addr: addr}, nil
}

// Example shows how to use custom dialer.
//
//    type MyConn struct {
//        net.Conn
//
//        addr string
//    }
//
//    func (cn MyConn) Hello() string {
//        return fmt.Sprintf("Hello from %s", cn.addr)
//    }
//
//    type MyDialer struct{}
//
//    func (MyDialer) Dial(ctx context.Context, addr string) (net.Conn, error) {
//        return MyConn{addr: addr}, nil
//    }
//
func ExampleNewConnPool_dialer() {
	p := NewConnPool(Config{
		Dialer: MyDialer{},
	})

	p.RegisterServer("google.com")

	cn, _ := p.OpenConn(context.Background())
	origCn := cn.OriginalConn().(MyConn)
	defer cn.ReturnToPool()

	// Use origCn in some way
	fmt.Println(origCn.Hello())

	// Output: Hello from google.com
}
