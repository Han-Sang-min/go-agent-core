//go:build linux

package collector

import (
	"io"
	"os"
)

func readProcCount() (int, bool) {
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
