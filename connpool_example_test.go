package goconnpool

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	net "net"
	"net/http"
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
			cn.MarkBroken()
		}

		// This call moves a connection back to pool or closes the connection when MarkBroken was called.
		cn.Close()
	}
}

func ExampleNewConnPool_httpRequest() {
	cfg := NewConfig(&flag.FlagSet{})
	flag.Parse() // XXX: This call is required to fill config variables

	// Create a pool
	pool := NewConnPool(*cfg)

	// Register some servers
	pool.RegisterServer("127.0.0.1:1234")

	cn, _ := pool.OpenConn(context.Background()) // success connection
	defer cn.Close()

	// You could implement your own transport in the same way:
	// https://golang.org/pkg/net/http/#RoundTripper
	req, _ := http.NewRequest(http.MethodGet, "/some", nil)
	req.Write(cn)
	resp, _ := http.ReadResponse(bufio.NewReader(cn), req)

	fmt.Println(resp.ContentLength)
}

func ExampleNewConnPool_blockingCalls() {
	cfg := NewConfig(&flag.FlagSet{})
	flag.Parse() // XXX: This call is required to fill config variables

	// Create a pool
	pool := NewConnPool(*cfg)

	// Register some servers
	pool.RegisterServer("127.0.0.1:1111")
	pool.RegisterServer("127.0.0.1:2222")

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
}

func (MyConn) Hello() string {
	return "Hello"
}

type MyDialer struct{}

func (MyDialer) Dial(ctx context.Context, addr string) (net.Conn, error) {
	return MyConn{}, nil
}

// Example shows how to use custom dialer.
//
//  type MyDialer struct{}
//
//  type MyConn struct {
//    net.Conn
//  }
//
//  func (MyConn) Hello() string {
//    return "Hello"
//  }
//
//  func (MyDialer) Dial(ctx context.Context, addr string) (net.Conn, error) {
//    return MyConn{}, nil
//  }
//
func ExampleNewConnPool_dialer() {
	p := NewConnPool(Config{
		Dialer: MyDialer{},
	})

	p.RegisterServer("google.com")

	cn, _ := p.OpenConn(context.Background())
	origCn := cn.OriginalConn().(MyConn)
	defer cn.Close() // XXX: Not origCn.Close() !!!

	// Use origCn in some way
	fmt.Println(origCn.Hello())

	// Output: Hello
}
