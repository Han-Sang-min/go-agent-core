//go:build linux

package collector

import (
	"context"
	"sync"
	"time"
)

type ContainerEnv struct {
	CommonEnv

	r *CgroupV2Reader

	mu        sync.Mutex
	prevTS    time.Time
	prevUsage uint64
	hasPrev   bool
}

func NewContainerEnv(r *CgroupV2Reader) *ContainerEnv {
	return &ContainerEnv{r: r}
}

func (e *ContainerEnv) Kind() string { return "Container" }

func (e *ContainerEnv) Mem(ctx context.Context) (MemInfo, error) {
	used, err := e.r.MemCurrent()
	if err != nil {
		return MemInfo{}, err
	}

	limit, unlimited, err := e.r.MemMax()
	if err != nil {
		return MemInfo{}, err
	}

	var percent float64
	var limitOut uint64
	if !unlimited && limit > 0 {
		limitOut = limit
		percent = (float64(used) / float64(limit)) * 100.0
	}

	return MemInfo{
		UsedBytes:   used,
		LimitBytes:  limitOut,
		UsedPercent: percent,
	}, nil
}

func (e *ContainerEnv) CPU(ctx context.Context) (CPUInfo, error) {
	usageUsec, err := e.r.CPUUsageUsec()
	if err != nil {
		return CPUInfo{}, err
	}

	quota, period, unlimited, err := e.r.CPUMax()
	if err != nil {
		return CPUInfo{}, err
	}

	limitCores := -1.0
	if !unlimited && period > 0 {
		limitCores = float64(quota) / float64(period)
		if limitCores <= 0 {
			limitCores = -1.0
		}
	}

	now := time.Now()

	e.mu.Lock()
	defer e.mu.Unlock()

	if !e.hasPrev {
		e.hasPrev = true
		e.prevTS = now
		e.prevUsage = usageUsec
		return CPUInfo{UsagePercent: 0, LimitCores: limitCores}, nil
	}

	dt := now.Sub(e.prevTS)
	du := usageUsec - e.prevUsage

	e.prevTS = now
	e.prevUsage = usageUsec

	wallUsec := float64(dt.Microseconds())
	if wallUsec <= 0 {
		return CPUInfo{UsagePercent: 0, LimitCores: limitCores}, nil
	}

	rawPercent := (float64(du) / wallUsec) * 100.0

	return CPUInfo{
		UsagePercent: rawPercent,
		LimitCores:   limitCores,
	}, nil
}
