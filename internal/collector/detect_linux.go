//go:build linux

package collector

import (
	"bufio"
	"errors"
	"os"
	"path/filepath"
	"strings"
)

func DetectEnv() RuntimeEnv {
	if isContainer() && isCgroupV2() {
		if cgPath, err := selfCgroupPathV2(); err == nil {
			reader := NewCgroupV2Reader(filepath.Join("/sys/fs/cgroup", cgPath))
			return NewContainerEnv(reader)
		}
	}
	return NewHostEnv()
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

func isCgroupV2() bool {
	return fileExists("/sys/fs/cgroup.controllers")
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
