package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"go-agent/internal/collector"
)

func main() {
	cfg := collector.DefaultConfig()

	flag.StringVar(&cfg.ListenAddr, "listen", cfg.ListenAddr, "gRPC listen address")
	flag.Parse()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigCh
		log.Printf("signal received: %v", sig)
		cancel()
	}()

	app := collector.New(cfg)
	if err := app.Run(ctx); err != nil {
		log.Fatalf("collector exited with error: %v", err)
	}
}
