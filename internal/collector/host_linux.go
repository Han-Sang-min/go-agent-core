//go:build linux

package collector

import (
	"bufio"
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

type HostEnv struct {
	CommonEnv

	hasPrev  bool
	prevStat hostCpuSample
}

type hostCpuSample struct {
	user, nice, system, idle, iowait, irq, softirq, steal uint64
	total                                                 uint64
	valid                                                 bool
}

type hostMemSample struct {
	totalKB     uint64
	availableKB uint64
	valid       bool
}

func NewHostEnv(root string) *HostEnv {
	if root == "" {
		root = "/"
	}
	return &HostEnv{
		CommonEnv: CommonEnv{procRoot: root}}
}

func (e *HostEnv) Kind() string {
	return "host"
}

func (e *HostEnv) CPU(ctx context.Context) (CPUStats, error) {
	var ret CPUStats

	curr := e.readCPU()

	prev := e.prevStat
	hasPrev := e.hasPrev
	e.prevStat = curr
	e.hasPrev = curr.valid

	if !hasPrev || !curr.valid || !prev.valid {
		return ret, nil
	}

	percent, ok := e.calcCpuUsage(prev, curr)
	if !ok {
		return ret, nil
	}

	ret.UsagePercent = percent
	ret.LimitCores = float64(runtime.NumCPU())
	ret.Valid = true

	return ret, nil
}

func (e *HostEnv) Mem(ctx context.Context) (MemStats, error) {
	var ret MemStats

	curr := e.readMem()
	if !curr.valid {
		return ret, nil
	}

	ret.UsedBytes = (curr.totalKB - curr.availableKB) * 1024
	ret.LimitBytes = curr.totalKB * 1024

	percent, ok := e.calcMemUsagePercent(curr)
	if !ok {
		return ret, nil
	}

	ret.UsedPercent = percent
	ret.Valid = true

	return ret, nil
}

func (e *HostEnv) readMem() hostMemSample {
	f, err := os.Open(filepath.Join(e.procRoot, "proc", "meminfo"))
	if err != nil {
		return hostMemSample{}
	}
	defer f.Close()

	sc := bufio.NewScanner(f)

	var total, avail uint64
	var haveTotal, haveAvail bool

	for sc.Scan() {
		line := sc.Text()
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		key := strings.TrimSuffix(fields[0], ":")
		v, err := strconv.ParseUint(fields[1], 10, 64)
		if err != nil {
			continue
		}

		switch key {
		case "MemTotal":
			total = v
			haveTotal = true
		case "MemAvailable":
			avail = v
			haveAvail = true
		}
	}

	if !haveTotal || !haveAvail || total == 0 || avail > total {
		return hostMemSample{}
	}

	return hostMemSample{
		totalKB:     total,
		availableKB: avail,
		valid:       true,
	}
}

func (e *HostEnv) calcMemUsagePercent(s hostMemSample) (float64, bool) {
	if !s.valid || s.totalKB == 0 || s.availableKB > s.totalKB {
		return 0, false
	}
	used := s.totalKB - s.availableKB
	usage := float64(used) / float64(s.totalKB) * 100.0
	if usage < 0.0 || usage > 100.0 {
		return 0, false
	}
	return usage, true
}

func (e *HostEnv) readCPU() hostCpuSample {
	f, err := os.Open(filepath.Join(e.procRoot, "proc", "stat"))
	if err != nil {
		return hostCpuSample{}
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	if !sc.Scan() {
		return hostCpuSample{}
	}
	line := sc.Text()
	fields := strings.Fields(line)
	if len(fields) < 5 || fields[0] != "cpu" {
		return hostCpuSample{}
	}

	parse := func(i int) (uint64, bool) {
		if len(fields) <= i {
			return 0, false
		}
		v, err := strconv.ParseUint(fields[i], 10, 64)
		if err != nil {
			return 0, false
		}
		return v, true
	}

	var ok bool
	var s hostCpuSample
	if s.user, ok = parse(1); !ok {
		return hostCpuSample{}
	}
	if s.nice, ok = parse(2); !ok {
		return hostCpuSample{}
	}
	if s.system, ok = parse(3); !ok {
		return hostCpuSample{}
	}
	if s.idle, ok = parse(4); !ok {
		return hostCpuSample{}
	}
	if s.iowait, ok = parse(5); !ok {
		return hostCpuSample{}
	}
	if s.irq, ok = parse(6); !ok {
		return hostCpuSample{}
	}
	if s.softirq, ok = parse(7); !ok {
		return hostCpuSample{}
	}
	if s.steal, ok = parse(8); !ok {
		return hostCpuSample{}
	}

	s.total = s.user + s.nice + s.system + s.idle + s.iowait + s.irq + s.softirq + s.steal
	s.valid = true
	return s
}

func (e *HostEnv) calcCpuUsage(prev, curr hostCpuSample) (float64, bool) {
	if !prev.valid || !curr.valid {
		return 0, false
	}

	if curr.total < prev.total {
		return 0, false
	}

	totalDelta := curr.total - prev.total

	prevIdle := prev.idle + prev.iowait
	currIdle := curr.idle + curr.iowait
	if currIdle < prevIdle {
		return 0, false
	}
	idleDelta := currIdle - prevIdle

	if totalDelta == 0 {
		return 0, false
	}

	usage := 1.0 - (float64(idleDelta) / float64(totalDelta))
	if usage < 0 || usage > 1 {
		return 0, false
	}

	return usage * 100.0, true
}
