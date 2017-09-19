package main

import "github.com/koding/tunnel"

func main() {
	cfg := &tunnel.ClientConfig{
		Identifier: "1234",
		ServerAddr: "203.0.113.0:80",
	}

	client, err := tunnel.NewClient(cfg)
	if err != nil {
		panic(err)
	}

	client.Start()
}
