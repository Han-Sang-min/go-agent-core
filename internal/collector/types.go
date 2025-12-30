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

type RuntimeEnv interface {
	Kind() string
	CPU(ctx context.Context) (CPUInfo, error)
	Mem(ctx context.Context) (MemInfo, error)
}
