package simulator

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"go-agent/internal/collector"
)

// Config holds the simulator's runtime configuration.
type Config struct {
	AgentCount    int
	CollectorAddr string
	Duration      time.Duration
	ScenarioName  string
	Interval      time.Duration
}

// Simulator orchestrates the embedded collector and N agents.
type Simulator struct {
	cfg    Config
	agents []*AgentRunner
	envs   []*SimulatedEnv
	dash   *Dashboard

	collectorAddr   string
	collectorCancel context.CancelFunc
	collectorWg     sync.WaitGroup
}

func New(cfg Config) *Simulator {
	if cfg.Interval == 0 {
		cfg.Interval = 1 * time.Second
	}
	return &Simulator{cfg: cfg}
}

func (s *Simulator) Run(ctx context.Context) error {
	scenarios := PredefinedScenarios()
	scenario, ok := scenarios[s.cfg.ScenarioName]
	if !ok {
		return fmt.Errorf("unknown scenario %q, available: %v", s.cfg.ScenarioName, scenarioNames(scenarios))
	}

	// Start embedded collector if no external address provided
	s.collectorAddr = s.cfg.CollectorAddr
	if s.collectorAddr == "" {
		s.collectorAddr = "localhost:50051"
		s.startCollector(ctx)
		defer s.stopCollector()
	}

	// Create simulated environments and agent runners
	s.envs = make([]*SimulatedEnv, s.cfg.AgentCount)
	s.agents = make([]*AgentRunner, s.cfg.AgentCount)
	for i := 0; i < s.cfg.AgentCount; i++ {
		s.envs[i] = NewSimulatedEnv(i, uint64(i*1000+1))
		s.agents[i] = NewAgentRunner(i, s.envs[i], s.collectorAddr, s.cfg.Interval)
	}

	s.dash = NewDashboard(s.agents)

	// Start all agents
	agentCtx, agentCancel := context.WithCancel(ctx)
	defer agentCancel()

	var agentWg sync.WaitGroup
	for _, a := range s.agents {
		agentWg.Add(1)
		go func(runner *AgentRunner) {
			defer agentWg.Done()
			runner.Run(agentCtx)
		}(a)
	}

	// Wait for agents to register
	time.Sleep(1 * time.Second)

	// Start dashboard
	dashCtx, dashCancel := context.WithCancel(ctx)
	defer dashCancel()
	go s.dash.Run(dashCtx)

	// Run scenario timeline
	s.runScenario(ctx, scenario)

	// Shutdown
	fmt.Println("\n[simulator] Scenario complete. Shutting down agents...")
	agentCancel()
	agentWg.Wait()
	dashCancel()

	s.dash.PrintSummary()

	return nil
}

func (s *Simulator) startCollector(parentCtx context.Context) {
	ctx, cancel := context.WithCancel(parentCtx)
	s.collectorCancel = cancel

	cfg := collector.DefaultConfig()
	cfg.ListenAddr = ":50051"
	cfg.StatePath = "" // no file persistence in simulator
	app := collector.New(cfg)

	s.collectorWg.Add(1)
	go func() {
		defer s.collectorWg.Done()
		if err := app.Run(ctx); err != nil {
			log.Printf("[simulator] collector error: %v", err)
		}
	}()

	// Wait for collector to bind the port
	time.Sleep(500 * time.Millisecond)
}

func (s *Simulator) stopCollector() {
	if s.collectorCancel != nil {
		s.collectorCancel()
		s.collectorWg.Wait()
		s.collectorCancel = nil
	}
}

func (s *Simulator) restartCollector(parentCtx context.Context) {
	log.Printf("[simulator] Restarting collector...")
	s.stopCollector()
	time.Sleep(300 * time.Millisecond) // wait for port release
	s.collectorWg = sync.WaitGroup{}
	s.startCollector(parentCtx)
	log.Printf("[simulator] Collector restarted — agents will re-register")
}

func (s *Simulator) runScenario(ctx context.Context, sc Scenario) {
	deadline := time.After(s.cfg.Duration)

	for _, phase := range sc.Phases {
		s.dash.SetPhase(phase.Name)
		log.Printf("[simulator] Phase: %s (duration=%s, mode=%s, network_down=%v, collector_restart=%v)",
			phase.Name, phase.Duration, phase.Mode, phase.NetworkDown, phase.CollectorRestart)

		if phase.CollectorRestart && s.cfg.CollectorAddr == "" {
			s.restartCollector(ctx)
		}

		targets := phase.TargetAgents
		if len(targets) == 0 {
			for i, env := range s.envs {
				env.SetMode(phase.Mode)
				s.agents[i].SetNetworkDown(phase.NetworkDown)
			}
		} else {
			for _, idx := range targets {
				if idx >= 0 && idx < len(s.envs) {
					s.envs[idx].SetMode(phase.Mode)
					s.agents[idx].SetNetworkDown(phase.NetworkDown)
				}
			}
		}

		phaseTimer := time.NewTimer(phase.Duration)
		select {
		case <-ctx.Done():
			phaseTimer.Stop()
			return
		case <-deadline:
			phaseTimer.Stop()
			log.Printf("[simulator] Duration limit reached during phase %q", phase.Name)
			return
		case <-phaseTimer.C:
		}
	}
}

func scenarioNames(m map[string]Scenario) []string {
	names := make([]string, 0, len(m))
	for k := range m {
		names = append(names, k)
	}
	return names
}
