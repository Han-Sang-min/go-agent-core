package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"go-agent/internal/simulator"
)

func main() {
	agents := flag.Int("agents", 5, "number of simulated agents")
	collectorAddr := flag.String("collector-addr", "", "external collector address (default: start embedded on :50051)")
	duration := flag.Duration("duration", 30*time.Second, "total simulation duration")
	scenario := flag.String("scenario", "full", "scenario to run (full, cpu-spike, mem-spike, network-fail, error-inject, stress)")
	interval := flag.Duration("interval", 1*time.Second, "agent collection interval")
	listScenarios := flag.Bool("list-scenarios", false, "list available scenarios and exit")
	flag.Parse()

	if *listScenarios {
		fmt.Println("Available scenarios:")
		for name, sc := range simulator.PredefinedScenarios() {
			phases := make([]string, len(sc.Phases))
			for i, p := range sc.Phases {
				phases[i] = p.Name
			}
			fmt.Printf("  %-15s  phases: %s\n", name, strings.Join(phases, " -> "))
		}
		os.Exit(0)
	}

	cfg := simulator.Config{
		AgentCount:    *agents,
		CollectorAddr: *collectorAddr,
		Duration:      *duration,
		ScenarioName:  *scenario,
		Interval:      *interval,
	}

	log.Printf("Starting simulator: agents=%d scenario=%s duration=%s interval=%s",
		cfg.AgentCount, cfg.ScenarioName, cfg.Duration, cfg.Interval)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigCh
		log.Printf("Signal received: %v, shutting down...", sig)
		cancel()
	}()

	sim := simulator.New(cfg)
	if err := sim.Run(ctx); err != nil {
		log.Fatalf("simulator error: %v", err)
	}
}
