//go:build linux

package env

import (
	"bufio"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
)

type CPUInfo struct {
	UsagePercent float64
	LimitCores   float64
}

type MemInfo struct {
	UsedBytes   uint64
	LimitBaytes uint64
	UsedPercent uint64
}

type RuntimeEnv interface {
	Kind() string
	CPU(ctx context.Context) (CPUInfo, error)
	MEM(ctx context.Context) (MemInfo, error)
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
