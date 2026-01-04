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

	config := flag.String("config", "", "config file path")
	once := flag.Bool("once", false, "run once")
	flag.Parse()

	_ = config

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(sigCh)

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	fmt.Print("Agent Start.\n")

	collect := func() {
		seq := counter.Add(1)
		cpuInfo, err := env.CPU(ctx)
		if err != nil {
			fmt.Printf("CPU Error: %v\n", err)
		}

		memInfo, err := env.Mem(ctx)
		if err != nil {
			fmt.Printf("Mem Error: %v\n", err)
		}

		diskInfo, err := env.Disk(ctx)
		if err != nil {
			fmt.Printf("Disk Error: %v\n", err)
		}

		procInfo, err := env.Procs(ctx)
		if err != nil {
			fmt.Printf("Proc Error: %v\n", err)
		}

		fmt.Printf("[Seq: %d] [Time: %s] CPU: %.2f%%, Mem: %.2f%%, Disk: %.2f%%, Procs: %d\n",
			seq,
			time.Now().In(loc),
			cpuInfo.UsagePercent,
			memInfo.UsedPercent, diskInfo.UsedPercent, procInfo.Count)
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
