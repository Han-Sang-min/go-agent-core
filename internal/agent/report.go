package agent

import (
	"context"
	"fmt"
	"math"
	"sync/atomic"
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

func ConsolOut(ctx context.Context, env RuntimeEnv) {
	seq := counter.Add(1)
	cpuStats, err := env.CPU(ctx)
	if err != nil {
		fmt.Printf("CPU Error: %v\n", err)
	}

	memStats, err := env.Mem(ctx)
	if err != nil {
		fmt.Printf("Mem Error: %v\n", err)
	}

	diskStats, err := env.Disk(ctx)
	if err != nil {
		fmt.Printf("Disk Error: %v\n", err)
	}

	ProcStats, err := env.Procs(ctx)
	if err != nil {
		fmt.Printf("Proc Error: %v\n", err)
	}

	cpuStr := "    N/A"
	if cpuStats.Valid {
		cpuStr = fmt.Sprintf("%7.2f%%", cpuStats.UsagePercent)
	}

	memStr := "    N/A"
	if memStats.Valid {
		if math.IsNaN(memStats.UsedPercent) {
			memStr = fmt.Sprintf(" N/A (%s)", formatBytes(memStats.UsedBytes))
		} else {
			memStr = fmt.Sprintf("%7.2f%%", memStats.UsedPercent)
		}
	}

	diskStr := "    N/A"
	if diskStats.Valid {
		diskStr = fmt.Sprintf("%7.2f%%", diskStats.UsedPercent)
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
