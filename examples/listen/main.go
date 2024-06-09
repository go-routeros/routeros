package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/go-routeros/routeros/v3"
)

var (
	debug    = flag.Bool("debug", false, "debug log level mode")
	command  = flag.String("command", "/ip/firewall/address-list/listen", "RouterOS command")
	address  = flag.String("address", "127.0.0.1:8728", "RouterOS address and port")
	username = flag.String("username", "admin", "User name")
	password = flag.String("password", "admin", "Password")
	timeout  = flag.Duration("timeout", 10*time.Second, "Cancel after")
	async    = flag.Bool("async", false, "Call Async()")
	useTLS   = flag.Bool("tls", false, "Use TLS")
)

func dial(ctx context.Context) (*routeros.Client, error) {
	if *useTLS {
		return routeros.DialTLSContext(ctx, *address, *username, *password, nil)
	}
	return routeros.DialContext(ctx, *address, *username, *password)
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

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	var cli *routeros.Client
	if cli, err = dial(ctx); err != nil {
		fatal(log, "could not dial", err)
	}

	cli.SetLogHandler(handler)
	defer func() {
		if errClose := cli.Close(); errClose != nil {
			log.Error("could not close routerOS client", slog.Any("error", errClose))
		}
	}()

	cli.Queue = 100

	if *async {
		explicitAsync(ctx, log, cli)
	} else {
		implicitAsync(ctx, log, cli)
	}

	if err = cli.Close(); err != nil {
		fatal(log, "could not close client", err)
	}
}

func explicitAsync(ctx context.Context, log *slog.Logger, c *routeros.Client) {
	errC := c.Async()
	log.Debug("Running explicitAsync mode...")

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	go func() {
		l, err := c.ListenArgsContext(ctx, strings.Split(*command, " "))
		if err != nil {
			fatal(log, "could not listen", err)
		}

		go func() {
			time.Sleep(*timeout)

			log.Debug("Cancelling the RouterOS command...")

			if _, errCancel := l.CancelContext(ctx); errCancel != nil {
				fatal(log, "could not cancel context", errCancel)
			}

			log.Debug("cancelled")
			cancel()
		}()

		log.Info("Waiting for !re...")
		for sen := range l.Chan() {
			log.Info("Update", slog.String("sentence", sen.String()))
		}

		if err = l.Err(); err != nil {
			fatal(log, "received an error", err)
		}
	}()

	select {
	case <-ctx.Done():
		return
	case err := <-errC:
		if err != nil {
			fatal(log, "received an error", err)
		}
	}
}

func implicitAsync(ctx context.Context, log *slog.Logger, c *routeros.Client) {
	l, err := c.ListenArgsContext(ctx, strings.Split(*command, " "))
	if err != nil {
		fatal(log, "could not listen", err)
	}

	go func() {
		time.Sleep(*timeout)
		log.Debug("Cancelling the RouterOS command...")
		if _, errCancel := l.Cancel(); errCancel != nil {
			fatal(log, "could not cancel", errCancel)
		}
	}()

loop:
	for {
		select {
		case <-ctx.Done():
			break loop
		case sen, ok := <-l.Chan():
			if !ok {
				break loop
			}

			log.Info("Update", slog.String("sentence", sen.String()))
		}
	}
	if err = l.Err(); err != nil {
		fatal(log, "received an error", err)
	}
}
