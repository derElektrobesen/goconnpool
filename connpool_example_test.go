package goconnpool

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"net/http"
	"time"
)

func ExampleNewConnPool_base() {
	cfg := NewConfig(&flag.FlagSet{})
	flag.Parse() // XXX: This call is required to fill config variables

	// Create a pool
	pool := NewConnPool(*cfg)

	// Register some servers
	pool.RegisterServer("tcp", "127.0.0.1:1234")
	pool.RegisterServer("tcp", "8.8.8.8:1234")

	for i := 0; i < 10; i++ {
		cn, err := pool.OpenConn(context.Background()) // Context could be cancelable here
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
	pool.RegisterServer("tcp", "127.0.0.1:1234")

	cn, _ := pool.OpenConn(context.Background())
	defer cn.Close()

	// You could implement your own transport in the same way:
	// https://golang.org/pkg/net/http/#RoundTripper
	req, _ := http.NewRequest(http.MethodGet, "/some", nil)
	req.Write(cn)
	resp, _ := http.ReadResponse(bufio.NewReader(cn), req)

	fmt.Println(resp.ContentLength)
}
