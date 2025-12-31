//go:build linux

package collector

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type CgroupV2Reader struct {
	base string
}

func NewCgroupV2Reader(base string) *CgroupV2Reader {
	return &CgroupV2Reader{base: base}
}

func (r *CgroupV2Reader) readFile(name string) (string, error) {
	b, err := os.ReadFile(filepath.Join(r.base, name))
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(b)), nil
}

func (r *CgroupV2Reader) MemCurrent() (uint64, error) {
	s, err := r.readFile("memory.current")
	if err != nil {
		return 0, err
	}
	return parseUint(s)
}

func (r *CgroupV2Reader) MemMax() (limit uint64, unlimited bool, err error) {
	s, err := r.readFile("memory.max")
	if err != nil {
		return 0, false, err
	}
	if s == "max" {
		return 0, true, nil
	}
	v, err := parseUint(s)
	return v, false, err
}

func (r *CgroupV2Reader) CPUUsageUsec() (uint64, error) {
	s, err := r.readFile("cpu.stat")
	if err != nil {
		return 0, err
	}
	for _, line := range strings.Split(s, "\n") {
		fields := strings.Fields(line)
		if len(fields) == 2 && fields[0] == "usage_usec" {
			return parseUint(fields[1])
		}
	}
	return 0, errors.New("usage_usec not found in cpu.stat")
}

func (r *CgroupV2Reader) CPUMax() (quotaUsec uint64, periodUsec uint64, unlimited bool, err error) {
	s, err := r.readFile("cpu.max")
	if err != nil {
		return 0, 0, false, err
	}

	fields := strings.Fields(s)
	if len(fields) != 2 {
		return 0, 0, false, fmt.Errorf("unexpected cpu.max format: %q", s)
	}

	period, err := parseUint(fields[1])
	if err != nil {
		return 0, 0, false, err
	}

	if fields[0] == "max" {
		return 0, period, true, nil
	}

	quota, err := parseUint(fields[0])
	if err != nil {
		return 0, 0, false, err
	}

	return quota, period, false, nil
}
