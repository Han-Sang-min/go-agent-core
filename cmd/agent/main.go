package main

import (
	"context"
	"flag"
	"fmt"
	"go-agent/internal/collector"
	"go-agent/internal/config"
	"math"
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

func formatBytes(b uint64) string {
	const (
		KB = 1024
		MB = 1024 * KB
		GB = 1024 * MB
	)

	switch {
	case b >= GB:
		return fmt.Sprintf("%.2fGiB", float64(b)/float64(GB))
	case b >= MB:
		return fmt.Sprintf("%.2fMiB", float64(b)/float64(MB))
	case b >= KB:
		return fmt.Sprintf("%.2fKiB", float64(b)/float64(KB))
	default:
		return fmt.Sprintf("%dB", b)
	}
}

func main() {
	configPath := flag.String("config", "", "config file path")
	once := flag.Bool("once", false, "run once")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "config load failed: %v\n", err)
		os.Exit(1)
	}

	var env collector.RuntimeEnv = collector.DetectEnv()
	fmt.Printf("Detected Environment: %s\n", env.Kind())
	fmt.Printf("Config interval: %s\n", cfg.Interval.Duration)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(sigCh)

	ticker := time.NewTicker(cfg.Interval.Duration)
	defer ticker.Stop()

	fmt.Print("Agent Start.\n")

	collect := func() {
		seq := counter.Add(1)
		CPUStats, err := env.CPU(ctx)
		if err != nil {
			fmt.Printf("CPU Error: %v\n", err)
		}

		MemStats, err := env.Mem(ctx)
		if err != nil {
			fmt.Printf("Mem Error: %v\n", err)
		}

		DiskStats, err := env.Disk(ctx)
		if err != nil {
			fmt.Printf("Disk Error: %v\n", err)
		}

		ProcStats, err := env.Procs(ctx)
		if err != nil {
			fmt.Printf("Proc Error: %v\n", err)
		}

		cpuStr := "    N/A"
		if CPUStats.Valid {
			cpuStr = fmt.Sprintf("%7.2f%%", CPUStats.UsagePercent)
		}

		memStr := fmt.Sprintf("%7.2f%%", MemStats.UsedPercent)
		if math.IsNaN(MemStats.UsedPercent) {
			memStr = fmt.Sprintf(" N/A (%s)", formatBytes(MemStats.UsedBytes))
		}

		diskStr := "    N/A"
		if DiskStats.Valid {
			diskStr = fmt.Sprintf("%7.2f%%", DiskStats.UsedPercent)
		}

		procStr := "    N/A"
		if ProcStats.Valid {
			procStr = fmt.Sprintf("%6d", ProcStats.Count)
		}

		ts := time.Now().In(loc).Format("2006-01-02 15:04:05.000 MST")
		fmt.Printf(
			"[Seq:%6d] [Time:%s] CPU:%8s  Mem:%-10s  Disk:%7s  Procs:%6s\n",
			seq, ts, cpuStr, memStr, diskStr, procStr,
		)

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
