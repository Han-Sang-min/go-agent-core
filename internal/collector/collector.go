package collector

import (
	"context"
)

type Metrics struct {
	CPUPercent     float64
	MemUsedBytes   uint64
	MemTotalBytes  uint64
	DiskUsedBytes  uint64
	DiskTotalBytes uint64
	ProcessCount   int
}

type Collector interface {
	Name() string
	Collector(ctx context.Context) (Metrics, error)
}
