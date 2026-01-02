package main

import (
	"context"
	"flag"
	"fmt"
	"go-agent/internal/collector"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"
)

var (
	loc = mustLoadLocation("Asia/Seoul")
)
var counter atomic.Int64

func mustLoadLocation(src string) *time.Location {
	l, err := time.LoadLocation(src)
	if err != nil {
		panic(err)
	}
	return l
}

func main() {
	var env collector.RuntimeEnv = collector.DetectEnv()
	fmt.Printf("Detected Environment: %s\n", env.Kind())

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(sigCh)

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	config := flag.String("config", "", "config file path")
	once := flag.Bool("once", false, "run once")

	flag.Parse()

	_ = config
	_ = once

	fmt.Print("Agent Start.\n")

	collect := func() {
		ctx := context.Background()

		cpuInfo, err := env.CPU(ctx)
		if err != nil {
			fmt.Printf("CPU Error: %v\n", err)
		}

		memInfo, err := env.Mem(ctx)
		if err != nil {
			fmt.Printf("Mem Error: %v\n", err)
		}

		fmt.Printf("[Time: %s] CPU: %.2f%%, Mem: %.2f%%\n",
			time.Now().Format(time.RFC3339),
			cpuInfo.UsagePercent,
			memInfo.UsedPercent)
	}

	if *once {
		collect()
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
			collect()
		}
	}
}
