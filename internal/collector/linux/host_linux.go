//go:build linux

package collector

import (
	"bufio"
	"context"
	"os"
	"strconv"
	"strings"
	"time"
)

type HostMeterics struct {
	TS time.Time

	CpuPercent float64
	MemPercent float64
}

type cpuStat struct {
	user, nice, system, idle, iowait, irq, softirq, steal uint64
	total                                                 uint64
	valid                                                 bool
}

type memStat struct {
	totalKB     uint64
	availableKB uint64
	valid       bool
}

func readMem() memStat {
	f, err := os.Open("/proc/meminfo")
	if err != nil {
		return memStat{}
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
		return memStat{}
	}

	return memStat{
		totalKB:     total,
		availableKB: avail,
		valid:       true,
	}
}

func calcMemUsagePercent(s memStat) (float64, bool) {
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

func readCPU() cpuStat {
	f, err := os.Open("/proc/stat")
	if err != nil {
		return cpuStat{}
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	if !sc.Scan() {
		return cpuStat{}
	}
	line := sc.Text()
	fields := strings.Fields(line)
	if len(fields) < 5 || fields[0] != "cpu" {
		return cpuStat{}
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
	var s cpuStat
	if s.user, ok = parse(1); !ok {
		return cpuStat{}
	}
	if s.nice, ok = parse(2); !ok {
		return cpuStat{}
	}
	if s.system, ok = parse(3); !ok {
		return cpuStat{}
	}
	if s.idle, ok = parse(4); !ok {
		return cpuStat{}
	}
	if s.iowait, ok = parse(5); !ok {
		return cpuStat{}
	}
	if s.irq, ok = parse(6); !ok {
		return cpuStat{}
	}
	if s.softirq, ok = parse(7); !ok {
		return cpuStat{}
	}
	if s.steal, ok = parse(8); !ok {
		return cpuStat{}
	}

	s.total = s.user + s.nice + s.system + s.idle + s.iowait + s.irq + s.softirq + s.steal
	s.valid = true
	return s
}

func calcCpuUsage(prev, curr cpuStat) (float64, bool) {
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

func Collector(ctx context.Context, out chan<- HostMeterics) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	var prevCPU cpuStat

	for {
		select {
		case <-ctx.Done():
			return

		case <-ticker.C:
			var m HostMeterics
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

			out <- m
		}
	}
}
