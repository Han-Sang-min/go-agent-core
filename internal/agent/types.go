package agent

import (
	"context"
)

type KubernetesMeta struct {
	Namespace string
	PodName   string
	PodUID    string
	PodIP     string

	NodeName string
	NodeUID  string

	NodeLabels map[string]string

	Valid bool
}

type K8sMetaProvider interface {
	K8sMeta(ctx context.Context) (KubernetesMeta, error)
}

type CPUStats struct {
	UsagePercent float64
	LimitCores   float64
	Valid        bool
}

type MemStats struct {
	UsedBytes   uint64
	LimitBytes  uint64
	UsedPercent float64

	Valid bool
}

type DiskStats struct {
	TotalBytes  uint64
	UsedBytes   uint64
	UsedPercent float64

	Valid bool
}

type ProcStats struct {
	Count int

	Valid bool
}

type RuntimeEnv interface {
	Kind() string
	CPU(ctx context.Context) (CPUStats, error)
	Mem(ctx context.Context) (MemStats, error)
	Disk(ctx context.Context) (DiskStats, error)
	Procs(ctx context.Context) (ProcStats, error)
}
