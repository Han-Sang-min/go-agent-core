package simulator

import (
	"context"
	"errors"
	"math/rand/v2"
	"sync"

	"go-agent/internal/agent"
)

// ScenarioMode controls what values SimulatedEnv returns.
type ScenarioMode int

const (
	ModeNormal   ScenarioMode = iota // Baseline: CPU ~15-25%, Mem ~40-50%, Disk ~60%, Procs ~120
	ModeCPUSpike                     // CPU jumps to 85-99%
	ModeMemSpike                     // Memory jumps to 90-98%
	ModeDiskFull                     // Disk hits 95-99%
	ModeErrorCPU                     // CPU() returns error
	ModeErrorAll                     // All methods return errors
)

func (m ScenarioMode) String() string {
	switch m {
	case ModeNormal:
		return "normal"
	case ModeCPUSpike:
		return "cpu-spike"
	case ModeMemSpike:
		return "mem-spike"
	case ModeDiskFull:
		return "disk-full"
	case ModeErrorCPU:
		return "error-cpu"
	case ModeErrorAll:
		return "error-all"
	default:
		return "unknown"
	}
}

// SimulatedEnv implements agent.RuntimeEnv with controllable values.
type SimulatedEnv struct {
	mu      sync.RWMutex
	mode    ScenarioMode
	agentID int
	rng     *rand.Rand
}

func NewSimulatedEnv(agentID int, seed uint64) *SimulatedEnv {
	return &SimulatedEnv{
		mode:    ModeNormal,
		agentID: agentID,
		rng:     rand.New(rand.NewPCG(seed, seed+1)),
	}
}

func (s *SimulatedEnv) SetMode(m ScenarioMode) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.mode = m
}

func (s *SimulatedEnv) Mode() ScenarioMode {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.mode
}

func (s *SimulatedEnv) Kind() string {
	return "simulated"
}

func (s *SimulatedEnv) CPU(ctx context.Context) (agent.CPUStats, error) {
	s.mu.Lock()
	mode := s.mode
	j := s.rng.Float64()
	s.mu.Unlock()

	switch mode {
	case ModeErrorCPU, ModeErrorAll:
		return agent.CPUStats{}, errors.New("simulated CPU read error")
	case ModeCPUSpike:
		return agent.CPUStats{
			UsagePercent: 85.0 + j*14.0,
			LimitCores:   4.0,
			Valid:        true,
		}, nil
	default:
		return agent.CPUStats{
			UsagePercent: 15.0 + j*10.0,
			LimitCores:   4.0,
			Valid:        true,
		}, nil
	}
}

func (s *SimulatedEnv) Mem(ctx context.Context) (agent.MemStats, error) {
	s.mu.Lock()
	mode := s.mode
	j := s.rng.Float64()
	s.mu.Unlock()

	if mode == ModeErrorAll {
		return agent.MemStats{}, errors.New("simulated Mem read error")
	}

	var usedPercent float64
	switch mode {
	case ModeMemSpike:
		usedPercent = 90.0 + j*8.0
	default:
		usedPercent = 40.0 + j*10.0
	}

	limitBytes := uint64(8 * 1024 * 1024 * 1024) // 8 GiB
	usedBytes := uint64(float64(limitBytes) * usedPercent / 100.0)

	return agent.MemStats{
		UsedBytes:   usedBytes,
		LimitBytes:  limitBytes,
		UsedPercent: usedPercent,
		Valid:       true,
	}, nil
}

func (s *SimulatedEnv) Disk(ctx context.Context) (agent.DiskStats, error) {
	s.mu.Lock()
	mode := s.mode
	j := s.rng.Float64()
	s.mu.Unlock()

	if mode == ModeErrorAll {
		return agent.DiskStats{}, errors.New("simulated Disk read error")
	}

	var usedPercent float64
	switch mode {
	case ModeDiskFull:
		usedPercent = 95.0 + j*4.0
	default:
		usedPercent = 55.0 + j*10.0
	}

	totalBytes := uint64(100 * 1024 * 1024 * 1024) // 100 GiB
	usedBytes := uint64(float64(totalBytes) * usedPercent / 100.0)

	return agent.DiskStats{
		TotalBytes:  totalBytes,
		UsedBytes:   usedBytes,
		UsedPercent: usedPercent,
		Valid:       true,
	}, nil
}

func (s *SimulatedEnv) Procs(ctx context.Context) (agent.ProcStats, error) {
	s.mu.Lock()
	mode := s.mode
	j := s.rng.Float64()
	s.mu.Unlock()

	if mode == ModeErrorAll {
		return agent.ProcStats{}, errors.New("simulated Procs read error")
	}

	count := 100 + int(j*40.0)
	if mode == ModeCPUSpike || mode == ModeMemSpike {
		count = 300 + int(j*100.0)
	}

	return agent.ProcStats{
		Count: count,
		Valid: true,
	}, nil
}
