package agent

import "context"

// RuntimeEnv는 그대로 유지하고, K8sMetaProvider만 추가로 붙이는 랩퍼
type EnvWithK8sMeta struct {
	base RuntimeEnv
	k8s  *KubernetesEnv
}

func NewEnvWithK8sMeta(base RuntimeEnv, k8s *KubernetesEnv) *EnvWithK8sMeta {
	return &EnvWithK8sMeta{base: base, k8s: k8s}
}

func (e *EnvWithK8sMeta) CPU(ctx context.Context) (CPUStats, error)    { return e.base.CPU(ctx) }
func (e *EnvWithK8sMeta) Mem(ctx context.Context) (MemStats, error)    { return e.base.Mem(ctx) }
func (e *EnvWithK8sMeta) Disk(ctx context.Context) (DiskStats, error)  { return e.base.Disk(ctx) }
func (e *EnvWithK8sMeta) Procs(ctx context.Context) (ProcStats, error) { return e.base.Procs(ctx) }

func (e *EnvWithK8sMeta) K8sMeta(ctx context.Context) (KubernetesMeta, error) {
	return e.k8s.K8sMeta(ctx)
}
