package simulator

import "time"

// Phase describes one segment of a scenario timeline.
type Phase struct {
	Name             string
	Duration         time.Duration
	Mode             ScenarioMode
	NetworkDown      bool
	CollectorRestart bool // restart the embedded collector at the start of this phase
	// TargetAgents specifies which agent indices are affected.
	// nil means ALL agents.
	TargetAgents []int
}

// Scenario is a named sequence of phases.
type Scenario struct {
	Name   string
	Phases []Phase
}

// PredefinedScenarios returns all built-in scenarios keyed by name.
func PredefinedScenarios() map[string]Scenario {
	return map[string]Scenario{
		"full":              fullScenario(),
		"cpu-spike":         cpuSpikeScenario(),
		"mem-spike":         memSpikeScenario(),
		"network-fail":      networkFailScenario(),
		"error-inject":      errorInjectScenario(),
		"stress":            stressScenario(),
		"collector-restart": collectorRestartScenario(),
	}
}

func fullScenario() Scenario {
	return Scenario{
		Name: "full",
		Phases: []Phase{
			{Name: "normal", Duration: 8 * time.Second, Mode: ModeNormal},
			{Name: "cpu-spike", Duration: 5 * time.Second, Mode: ModeCPUSpike},
			{Name: "recovery-1", Duration: 4 * time.Second, Mode: ModeNormal},
			{Name: "mem-spike", Duration: 5 * time.Second, Mode: ModeMemSpike},
			{Name: "recovery-2", Duration: 4 * time.Second, Mode: ModeNormal},
			{Name: "network-down", Duration: 5 * time.Second, Mode: ModeNormal, NetworkDown: true},
			{Name: "network-recover", Duration: 4 * time.Second, Mode: ModeNormal},
			{Name: "disk-full", Duration: 4 * time.Second, Mode: ModeDiskFull},
			{Name: "final-normal", Duration: 3 * time.Second, Mode: ModeNormal},
		},
	}
}

func cpuSpikeScenario() Scenario {
	return Scenario{
		Name: "cpu-spike",
		Phases: []Phase{
			{Name: "baseline", Duration: 5 * time.Second, Mode: ModeNormal},
			{Name: "spike", Duration: 10 * time.Second, Mode: ModeCPUSpike},
			{Name: "recovery", Duration: 5 * time.Second, Mode: ModeNormal},
		},
	}
}

func memSpikeScenario() Scenario {
	return Scenario{
		Name: "mem-spike",
		Phases: []Phase{
			{Name: "baseline", Duration: 5 * time.Second, Mode: ModeNormal},
			{Name: "spike", Duration: 10 * time.Second, Mode: ModeMemSpike},
			{Name: "recovery", Duration: 5 * time.Second, Mode: ModeNormal},
		},
	}
}

func networkFailScenario() Scenario {
	return Scenario{
		Name: "network-fail",
		Phases: []Phase{
			{Name: "connected", Duration: 5 * time.Second, Mode: ModeNormal},
			{Name: "disconnected", Duration: 8 * time.Second, Mode: ModeNormal, NetworkDown: true},
			{Name: "reconnected", Duration: 7 * time.Second, Mode: ModeNormal},
		},
	}
}

func errorInjectScenario() Scenario {
	return Scenario{
		Name: "error-inject",
		Phases: []Phase{
			{Name: "normal", Duration: 5 * time.Second, Mode: ModeNormal},
			{Name: "cpu-errors", Duration: 5 * time.Second, Mode: ModeErrorCPU},
			{Name: "all-errors", Duration: 5 * time.Second, Mode: ModeErrorAll},
			{Name: "recovery", Duration: 5 * time.Second, Mode: ModeNormal},
		},
	}
}

func collectorRestartScenario() Scenario {
	return Scenario{
		Name: "collector-restart",
		Phases: []Phase{
			{Name: "before-restart", Duration: 5 * time.Second, Mode: ModeNormal},
			{Name: "collector-down", Duration: 8 * time.Second, Mode: ModeNormal, CollectorRestart: true},
			{Name: "recovery", Duration: 7 * time.Second, Mode: ModeNormal},
		},
	}
}

func stressScenario() Scenario {
	return Scenario{
		Name: "stress",
		Phases: []Phase{
			{Name: "warmup", Duration: 3 * time.Second, Mode: ModeNormal},
			{Name: "cpu-spike", Duration: 3 * time.Second, Mode: ModeCPUSpike},
			{Name: "mem-spike", Duration: 3 * time.Second, Mode: ModeMemSpike},
			{Name: "disk-full", Duration: 3 * time.Second, Mode: ModeDiskFull},
			{Name: "net-fail", Duration: 3 * time.Second, Mode: ModeNormal, NetworkDown: true},
			{Name: "all-errors", Duration: 3 * time.Second, Mode: ModeErrorAll},
			{Name: "recovery", Duration: 3 * time.Second, Mode: ModeNormal},
		},
	}
}
