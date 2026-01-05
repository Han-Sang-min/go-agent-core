//go:build linux

package collector

import (
	"context"
	"math"
	"time"
)

type ContainerEnv struct {
	CommonEnv

	r *CgroupV2Reader

	prevTS    time.Time
	prevUsage uint64
	hasPrev   bool
}

type cgroupCpuSample struct {
	now       time.Time
	usageUsec uint64

	quota     uint64
	period    uint64
	unlimited bool

	valid bool
}

type cgroupMemSample struct {
	usedBytes  uint64
	limitBytes uint64
	unlimited  bool
	valid      bool
}

func NewContainerEnv(r *CgroupV2Reader, rootPath string) *ContainerEnv {
	if rootPath == "" {
		rootPath = "/"
	}
	return &ContainerEnv{
		CommonEnv: CommonEnv{procRoot: rootPath},
		r:         r}
}

func (e *ContainerEnv) Kind() string { return "Container" }

func (e *ContainerEnv) CPU(ctx context.Context) (CPUStats, error) {
	curr := time.Now()
	sample, err := e.readCgroupCPU(curr)
	if err != nil {
		return CPUStats{}, err
	}
	return e.calcCgroupCpu(sample)
}

func (e *ContainerEnv) Mem(ctx context.Context) (MemStats, error) {
	sample, err := e.readCgroupMem()
	if err != nil {
		return MemStats{}, err
	}
	return e.calcCgroupMem(sample)
}

func (e *ContainerEnv) calcCgroupMem(s cgroupMemSample) (MemStats, error) {
	percent := math.NaN()
	var limitOut uint64
	if !s.unlimited && s.limitBytes > 0 {
		limitOut = s.limitBytes
		percent = (float64(s.usedBytes) / float64(s.limitBytes)) * 100.0
	}
	return MemStats{
		UsedBytes:   s.usedBytes,
		LimitBytes:  limitOut,
		UsedPercent: percent,
	}, nil
}

func (e *ContainerEnv) calcCgroupCpu(s cgroupCpuSample) (CPUStats, error) {
	limitCores := -1.0
	if !s.unlimited && s.period > 0 {
		limitCores = float64(s.quota) / float64(s.period)
		if limitCores <= 0 {
			limitCores = -1.0
		}
	}

	if !e.hasPrev {
		e.hasPrev = true
		e.prevTS = s.now
		e.prevUsage = s.usageUsec
		return CPUStats{UsagePercent: 0, LimitCores: limitCores}, nil
	}

	dt := s.now.Sub(e.prevTS)

	if dt <= 0 {
		e.prevTS = s.now
		e.prevUsage = s.usageUsec
		return CPUStats{UsagePercent: 0, LimitCores: limitCores}, nil
	}
	if s.usageUsec < e.prevUsage {
		e.prevTS = s.now
		e.prevUsage = s.usageUsec
		return CPUStats{UsagePercent: 0, LimitCores: limitCores}, nil
	}

	du := s.usageUsec - e.prevUsage

	e.prevTS = s.now
	e.prevUsage = s.usageUsec

	wallUsec := float64(dt.Microseconds())
	if wallUsec <= 0 {
		return CPUStats{UsagePercent: 0, LimitCores: limitCores}, nil
	}

	rawPercent := (float64(du) / wallUsec) * 100.0

	return CPUStats{
		UsagePercent: rawPercent,
		LimitCores:   limitCores,
	}, nil
}

func (e *ContainerEnv) readCgroupMem() (cgroupMemSample, error) {
	used, err := e.r.MemCurrent()
	if err != nil {
		return cgroupMemSample{}, err
	}
	limit, unlimited, err := e.r.MemMax()
	if err != nil {
		return cgroupMemSample{}, err
	}
	return cgroupMemSample{
		usedBytes:  used,
		limitBytes: limit,
		unlimited:  unlimited,
		valid:      true,
	}, nil
}

func (e *ContainerEnv) readCgroupCPU(now time.Time) (cgroupCpuSample, error) {
	usageUsec, err := e.r.CPUUsageUsec()
	if err != nil {
		return cgroupCpuSample{}, err
	}

	quota, period, unlimited, err := e.r.CPUMax()
	if err != nil {
		return cgroupCpuSample{}, err
	}

	return cgroupCpuSample{
		now:       now,
		usageUsec: usageUsec,
		quota:     quota,
		period:    period,
		unlimited: unlimited,
		valid:     true,
	}, nil
}
