package main

import (
	"flag"
	"log"
	"strings"

	"gopkg.in/routeros.v1"
)

var (
	command  = flag.String("command", "/system/resource/print", "RouterOS command")
	address  = flag.String("address", "127.0.0.1:8728", "RouterOS address and port")
	username = flag.String("username", "admin", "User name")
	password = flag.String("password", "admin", "Password")
	async    = flag.Bool("async", false, "Use async code")
	useTLS   = flag.Bool("tls", false, "Use TLS")
)

func main() {
	flag.Parse()

	c := &routeros.Client{
		Address:  *address,
		Username: *username,
		Password: *password,
	}
	var err error
	if *useTLS {
		err = c.ConnectTLS(nil)
	} else {
		err = c.Connect()
	}
	if err != nil {
		log.Fatal(err)
	}
	defer c.Close()

	if *async {
		c.Async()
	}

	r, err := c.RunArgs(strings.Split(*command, " "))
	if err != nil {
		log.Fatal(err)
	}

	log.Print(r)
}
