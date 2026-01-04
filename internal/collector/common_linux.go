//go:build linux

package collector

import (
	"context"
	"io"
	"os"
	"syscall"
)

type CommonEnv struct {
	procRoot string
}

type diskStat struct {
	total uint64
	avail uint64
	valid bool
}

func (c *CommonEnv) Disk(ctc context.Context) (DiskInfo, error) {
	var ret DiskInfo

	stat := c.readDisk("/")
	if !stat.valid {
		return ret, nil
	}

	percent, ok := c.calcDiskUsagePercent(stat)
	if !ok {
		return ret, nil
	}

	ret.TotalBytes = stat.total
	ret.UsedBytes = stat.total - stat.avail
	ret.UsedPercent = percent

	return ret, nil
}

func (c *CommonEnv) Procs(ctc context.Context) (ProcInfo, error) {
	var ret ProcInfo
	count, ok := c.readProcCount()
	if !ok {
		return ret, nil
	}

	ret.Count = count

	return ret, nil
}

func (c *CommonEnv) readDisk(path string) diskStat {
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

func (c *CommonEnv) calcDiskUsagePercent(s diskStat) (float64, bool) {
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

func (c *CommonEnv) readProcCount() (int, bool) {
	d, err := os.Open("/proc")
	if err != nil {
		return 0, false
	}
	defer d.Close()

	count := 0
	for {
		names, err := d.Readdirnames(512)
		if err != nil {
			if err == io.EOF {
				break
			}
			break
		}
		for _, name := range names {
			ok := true
			for i := 0; i < len(name); i++ {
				c := name[i]
				if c < '0' || c > '9' {
					ok = false
					break
				}
			}
			if ok {
				count++
			}
		}
	}
	return count, true
}
