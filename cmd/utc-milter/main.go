package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/kruton/utc-milter/internal/server"
)

var version = "dev"

func main() {
	var (
		network         string
		socket          string
		socketMode      uint
		socketUser      string
		socketGroup     string
		shutdownTimeout time.Duration
		showVersion     bool
	)

	flag.StringVar(&network, "network", "unix", "listen network: unix, tcp, tcp4, or tcp6")
	flag.StringVar(&socket, "socket", "/run/utc-milter/utc-milter.sock", "listen socket path or TCP address")
	flag.UintVar(&socketMode, "socket-mode", 0o660, "Unix socket permissions, in octal")
	flag.StringVar(&socketUser, "socket-user", "", "Unix socket owner name or numeric UID")
	flag.StringVar(&socketGroup, "socket-group", "postfix", "Unix socket group name or numeric GID")
	flag.DurationVar(&shutdownTimeout, "shutdown-timeout", 10*time.Second, "graceful shutdown timeout")
	flag.BoolVar(&showVersion, "version", false, "print version and exit")
	flag.Parse()

	if showVersion {
		fmt.Println(version)
		return
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg := server.Config{
		Network:         network,
		Address:         socket,
		SocketMode:      os.FileMode(socketMode),
		SocketUser:      socketUser,
		SocketGroup:     socketGroup,
		ShutdownTimeout: shutdownTimeout,
		Logger:          log.New(os.Stdout, "", log.LstdFlags),
	}
	if err := server.Run(ctx, cfg); err != nil {
		log.Fatal(err)
	}
}
