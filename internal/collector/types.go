package collector

import (
	"context"
)

type CPUInfo struct {
	UsagePercent float64
	LimitCores   float64
}

type MemInfo struct {
	UsedBytes   uint64
	LimitBytes  uint64
	UsedPercent float64
}

type DiskInfo struct {
	TotalBytes  uint64
	UsedBytes   uint64
	UsedPercent float64
}

type ProcInfo struct {
	Count int
}

type RuntimeEnv interface {
	Kind() string
	CPU(ctx context.Context) (CPUInfo, error)
	Mem(ctx context.Context) (MemInfo, error)
	Disk(ctx context.Context) (DiskInfo, error)
	Procs(ctx context.Context) (ProcInfo, error)
}
