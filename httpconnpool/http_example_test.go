package httpconnpool

import (
	"flag"
	"fmt"
	"net/http"
)

func ExampleNewTransport() {
	cfg := NewConfig(&flag.FlagSet{})
	flag.Parse()

	tr := NewTransport(*cfg)
	tr.RegisterServer("google.com")

	cli := http.Client{
		Transport: tr,
	}

	req, err := http.NewRequest(http.MethodGet, "/search", nil)
	if err != nil {
		fmt.Println(err)
		return
	}

	response, err := cli.Do(req)
	if err != nil {
		fmt.Println(err)
	} else {
		fmt.Println(*response)
	}
}
