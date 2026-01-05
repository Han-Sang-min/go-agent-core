package collector

import (
	"context"
)

type CPUStats struct {
	UsagePercent float64
	LimitCores   float64
}

type MemStats struct {
	UsedBytes   uint64
	LimitBytes  uint64
	UsedPercent float64
}

type DiskStats struct {
	TotalBytes  uint64
	UsedBytes   uint64
	UsedPercent float64
}

type ProcStats struct {
	Count int
}

type RuntimeEnv interface {
	Kind() string
	CPU(ctx context.Context) (CPUStats, error)
	Mem(ctx context.Context) (MemStats, error)
	Disk(ctx context.Context) (DiskStats, error)
	Procs(ctx context.Context) (ProcStats, error)
}
