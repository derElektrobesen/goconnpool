package goconnpool

import (
	"context"
	"flag"
	"time"
)

func ExampleNewConnPool() {
	cfg := NewConfig(&flag.FlagSet{})
	flag.Parse()

	pool := NewConnPool(*cfg)
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
			cn.MarkUnusable()
		}

		cn.Close()
	}
}
