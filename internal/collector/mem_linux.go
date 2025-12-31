//go:build linux

package collector

import (
	"bufio"
	"os"
	"strconv"
	"strings"
)

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
