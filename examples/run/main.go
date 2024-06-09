package main

import (
	"flag"
	"log/slog"
	"os"
	"strings"

	"github.com/go-routeros/routeros/v3"
)

var (
	debug    = flag.Bool("debug", false, "debug log level mode")
	command  = flag.String("command", "/system/resource/print", "RouterOS command")
	address  = flag.String("address", "127.0.0.1:8728", "RouterOS address and port")
	username = flag.String("username", "admin", "User name")
	password = flag.String("password", "admin", "Password")
	async    = flag.Bool("async", false, "Use async code")
	useTLS   = flag.Bool("tls", false, "Use TLS")
)

func dial() (*routeros.Client, error) {
	if *useTLS {
		return routeros.DialTLS(*address, *username, *password, nil)
	}
	return routeros.Dial(*address, *username, *password)
}

func fatal(log *slog.Logger, message string, err error) {
	log.Error(message, slog.Any("error", err))
	os.Exit(2)
}

func main() {
	var err error
	if err = flag.CommandLine.Parse(os.Args[1:]); err != nil {
		panic(err)
	}

	logLevel := slog.LevelInfo
	if debug != nil && *debug {
		logLevel = slog.LevelDebug
	}

	handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		AddSource: true,
		Level:     logLevel,
	})

	log := slog.New(handler)

	c, err := dial()
	if err != nil {
		fatal(log, "could not connect", err)
	}
	defer c.Close()

	c.SetLogHandler(handler)

	if *async {
		c.Async()
	}

	r, err := c.RunArgs(strings.Split(*command, " "))
	if err != nil {
		fatal(log, "could not run args", err)
	}

	log.Info("received results", slog.Any("results", r))
}
