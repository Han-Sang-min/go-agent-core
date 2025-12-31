//go:build linux

package collector

import (
	"syscall"
)

type diskStat struct {
	total uint64
	avail uint64
	valid bool
}

func readDisk(path string) diskStat {
	var st syscall.Statfs_t
	if err := syscall.Statfs(path, &st); err != nil {
		return diskStat{}
	}

	bsize := uint64(st.Bsize)
	total := st.Blocks * bsize
	avail := st.Bavail * bsize

	if total == 0 || avail > total {
		return diskStat{}
	}
	return diskStat{total: total, avail: avail, valid: true}
}

func calcDiskUsagePercent(s diskStat) (float64, bool) {
	if !s.valid || s.total == 0 || s.avail > s.total {
		return 0, false
	}
	used := s.total - s.avail
	usage := float64(used) / float64(s.total) * 100.0
	if usage < 0 || usage > 100 {
		return 0, false
	}
	return usage, true
}
