//go:build linux

package env

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

type CPUInfo struct {
	UsagePercent float64
	LimitCores   float64
}

type MemInfo struct {
	UsedBytes   uint64
	LimitBytes  uint64
	UsedPercent float64
}

type RuntimeEnv interface {
	Kind() string
	CPU(ctx context.Context) (CPUInfo, error)
	Mem(ctx context.Context) (MemInfo, error)
}

func DetectEnv() RuntimeEnv {
	if isContainer() && isCgroupV2() {
		if cgPath, err := selfCgroupPathV2(); err == nil {
			reader := NewCgroupV2Reader(filepath.Join("/sys/fs/cgroup", cgPath))
			return NewContainerEnv(reader)
		}
	}
	return NewHostEnv()
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil || !os.IsNotExist(err)
}

func isCgroupV2() bool {
	return fileExists("/sys/fs/cgroup.controllers")
}

func isContainer() bool {
	if fileExists("/.dockerenv") || fileExists("/run/.containerenv") {
		return true
	}

	b, err := os.ReadFile("/proc/1/cgroup")
	if err == nil {
		s := string(b)
		keywords := []string{"docker", "kubepods", "containerd", "podman", "lxc"}
		for _, k := range keywords {
			if strings.Contains(s, k) {
				return true
			}
		}
	}
	return false
}

func selfCgroupPathV2() (string, error) {
	f, err := os.Open("/proc/self/cgroup")
	if err != nil {
		return "", err
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := sc.Text()
		parts := strings.SplitN(line, ":", 3)
		if len(parts) != 3 {
			continue
		}
		if parts[0] == "0" && parts[1] == "" {
			p := parts[2]
			if p == "" {
				return "", errors.New("empty cgroup path")
			}
			return p, nil
		}
	}
	if err := sc.Err(); err != nil {
		return "", err
	}
	return "", errors.New("cgroup v2 path not found in /proc/self/cgroup")
}

// ---------

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

func parseUint(s string) (uint64, error) {
	return strconv.ParseUint(strings.TrimSpace(s), 10, 64)
}

// -------

type ContainerEnv struct {
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

// -----

type HostEnv struct{}

func NewHostEnv() *HostEnv { return &HostEnv{} }

func (e *HostEnv) Kind() string {
	return "host"
}

func (e *HostEnv) CPU(ctx context.Context) (CPUInfo, error) {
	// TODO: Attech host CPU
	return CPUInfo{}, errors.New("HostEnv.CPU: not implemeted")
}

func (e *HostEnv) Mem(ctx context.Context) (MemInfo, error) {
	// TODO: Attect host Mem
	return MemInfo{}, errors.New("HostEnv.Mem: not implemeted")
}
