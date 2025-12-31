//go:build linux

package collector

import (
	"context"
	"time"
)

type HostMetrics struct {
	TS time.Time

	CpuPercent float64
	MemPercent float64

	DiskPercent float64
	ProcCount   int
}

func Collector(ctx context.Context, out chan<- HostMetrics) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	var prevCPU cpuStat

	for {
		select {
		case <-ctx.Done():
			return

		case <-ticker.C:
			var m HostMetrics
			m.TS = time.Now()

			// CPU
			currCPU := readCPU()
			if prevCPU.valid {
				if cpu, ok := calcCpuUsage(prevCPU, currCPU); ok {
					m.CpuPercent = cpu
				}
			}
			prevCPU = currCPU

			// MEM
			mem := readMem()
			if mem.valid {
				if memp, ok := calcMemUsagePercent(mem); ok {
					m.MemPercent = memp
				}
			}

			// DISK
			ds := readDisk("/")
			if ds.valid {
				if dp, ok := calcDiskUsagePercent(ds); ok {
					m.DiskPercent = dp
				}
			}

			// PROCESS
			if pc, ok := readProcCount(); ok {
				m.ProcCount = pc
			}

			out <- m
		}
	}
}
