package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/go-routeros/routeros/v3"
)

var (
	debug      = flag.Bool("debug", false, "debug log level mode")
	address    = flag.String("address", "192.168.0.1:8728", "Address")
	username   = flag.String("username", "admin", "Username")
	password   = flag.String("password", "admin", "Password")
	properties = flag.String("properties", "name,rx-byte,tx-byte,rx-packet,tx-packet", "Properties")
	interval   = flag.Duration("interval", 1*time.Second, "Interval")
)

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

	c, err := routeros.Dial(*address, *username, *password)
	if err != nil {
		fatal(log, "could not dial", err)
	}

	c.SetLogHandler(handler)

	for {
		var reply *routeros.Reply

		if reply, err = c.Run("/interface/print", "?disabled=false", "?running=true", "=.proplist="+*properties); err != nil {
			fatal(log, "could not run", err)
		}

		for _, re := range reply.Re {
			for _, p := range strings.Split(*properties, ",") {
				fmt.Print(re.Map[p], "\t")
			}
			fmt.Print("\n")
		}
		fmt.Print("\n")

		time.Sleep(*interval)
	}
}
