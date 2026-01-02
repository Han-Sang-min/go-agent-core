//go:build linux

package collector

import (
	"bufio"
	"os"
	"strconv"
	"strings"
)

type cpuStat struct {
	user, nice, system, idle, iowait, irq, softirq, steal uint64
	total                                                 uint64
	valid                                                 bool
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
