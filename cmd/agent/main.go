package main

import (
	"context"
	"flag"
	"fmt"
	"go-agent/internal/agent"
	"go-agent/internal/config"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	configPath := flag.String("config", "", "config file path")
	once := flag.Bool("once", false, "run once")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "config load failed: %v\n", err)
		os.Exit(1)
	}

	var env agent.RuntimeEnv = agent.DetectEnv()
	fmt.Printf("Detected Environment: %s\n", env.Kind())
	fmt.Printf("Config interval: %s\n", cfg.Interval.Duration)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	grpc, err := agent.NewGRPCOut(ctx, "127.0.0.1:50051")
	if err != nil {
		fmt.Fprintf(os.Stderr, "grpc agent load failed: %v\n", err)
		os.Exit(1)
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(sigCh)

	ticker := time.NewTicker(cfg.Interval.Duration)
	defer ticker.Stop()

	fmt.Print("Agent Start.\n")

	if *once {
		agent.ConsolOut(ctx, env)
		fmt.Println("Agent Stop.")
		return
	}

	for {
		select {
		case sig := <-sigCh:
			fmt.Printf("received: %v\n", sig)
			fmt.Println("Agent Stop.")
			return
		case <-ticker.C:
			agent.ConsolOut(ctx, env)
			grpc.SendHeartbeat(ctx)
		}
	}
}
