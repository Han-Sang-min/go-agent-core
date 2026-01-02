//go:build linux

package collector

import (
	"context"
	"errors"
)

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
