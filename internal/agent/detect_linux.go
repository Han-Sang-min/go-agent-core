//go:build linux

package agent

import (
	"bufio"
	"errors"
	"go-agent/internal/util"
	"os"
	"path/filepath"
	"strings"
)

func DetectEnv() RuntimeEnv {
	var base RuntimeEnv

	if isContainer() && isCgroupV2() {
		if cgPath, err := selfCgroupPathV2(); err == nil {
			cgPath = strings.TrimPrefix(cgPath, "/")
			cgBase := "/sys/fs/cgroup"
			if cgPath != "" {
				cgBase = filepath.Join(cgBase, cgPath)
			}
			reader := NewCgroupV2Reader(cgBase)
			base = NewContainerEnv(reader, "")
		}
	}
	if base == nil {
		base = NewHostEnv("")
	}

	if isKubernetes() {
		if k8s, err := NewKubernetesEnv(); err == nil {
			return NewEnvWithK8sMeta(base, k8s)
		}
	}

	return base
}

func isContainer() bool {
	if util.FileExists("/.dockerenv") || util.FileExists("/run/.containerenv") {
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
	return util.FileExists("/sys/fs/cgroup/cgroup.controllers")
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
