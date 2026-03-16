package simulator

import (
	"context"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"go-agent/internal/agent"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// AgentStatus is the observable state of one simulated agent.
type AgentStatus struct {
	ID          int
	AgentUUID   string
	Connected   bool
	NetworkDown bool
	LastCollect time.Time
	LastMetrics agent.Collected
	LastError   string
	SendCount   int64
	ErrorCount  int64
}

// AgentRunner manages a single simulated agent's lifecycle.
type AgentRunner struct {
	id       int
	env      *SimulatedEnv
	interval time.Duration
	addr     string

	mu     sync.RWMutex
	status AgentStatus

	networkDown atomic.Bool
	grpcOut     *agent.GRPCOut

	sendCount  atomic.Int64
	errorCount atomic.Int64
}

func NewAgentRunner(id int, env *SimulatedEnv, collectorAddr string, interval time.Duration) *AgentRunner {
	return &AgentRunner{
		id:       id,
		env:      env,
		interval: interval,
		addr:     collectorAddr,
		status:   AgentStatus{ID: id},
	}
}

// SetNetworkDown simulates network failure.
func (r *AgentRunner) SetNetworkDown(down bool) {
	r.networkDown.Store(down)
}

// Status returns a snapshot of the agent's current state.
func (r *AgentRunner) Status() AgentStatus {
	r.mu.RLock()
	defer r.mu.RUnlock()
	s := r.status
	s.SendCount = r.sendCount.Load()
	s.ErrorCount = r.errorCount.Load()
	s.NetworkDown = r.networkDown.Load()
	return s
}

// Run starts the agent's collect-send loop. Blocks until ctx is cancelled.
func (r *AgentRunner) Run(ctx context.Context) {
	r.connect(ctx)

	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()
	defer r.disconnect()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			r.tick(ctx)
		}
	}
}

func (r *AgentRunner) connect(ctx context.Context) {
	if r.networkDown.Load() {
		return
	}

	grpcOut, err := agent.NewGRPCOut(ctx, r.addr)
	if err != nil {
		r.mu.Lock()
		r.status.Connected = false
		r.status.LastError = fmt.Sprintf("connect: %v", err)
		r.mu.Unlock()
		r.errorCount.Add(1)
		return
	}
	r.grpcOut = grpcOut

	r.mu.Lock()
	r.status.Connected = true
	r.status.AgentUUID = grpcOut.AgentID()
	r.status.LastError = ""
	r.mu.Unlock()
}

func (r *AgentRunner) disconnect() {
	if r.grpcOut != nil {
		_ = r.grpcOut.Close()
		r.grpcOut = nil
	}
	r.mu.Lock()
	r.status.Connected = false
	r.mu.Unlock()
}

func (r *AgentRunner) tick(ctx context.Context) {
	netDown := r.networkDown.Load()

	// Network down: disconnect and skip
	if netDown && r.grpcOut != nil {
		r.disconnect()
		r.mu.Lock()
		r.status.LastError = "network down (simulated)"
		r.mu.Unlock()
		return
	}
	if netDown {
		return
	}

	// Network up but not connected: reconnect
	if !netDown && r.grpcOut == nil {
		r.connect(ctx)
		if r.grpcOut == nil {
			return
		}
	}

	// Collect metrics using real Collect() with our SimulatedEnv
	collected := agent.Collect(ctx, r.env)

	r.mu.Lock()
	r.status.LastCollect = time.Now()
	r.status.LastMetrics = collected
	r.mu.Unlock()

	if r.grpcOut == nil {
		return
	}

	// Send heartbeat
	res, err := r.grpcOut.SendHeartbeat(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			log.Printf("[sim-agent-%03d] not found on collector, re-registering...", r.id)
			r.disconnect()
			r.connect(ctx)
		} else {
			r.errorCount.Add(1)
			r.mu.Lock()
			r.status.LastError = fmt.Sprintf("heartbeat: %v", err)
			r.mu.Unlock()
		}
	} else {
		r.mu.Lock()
		r.status.LastError = ""
		r.mu.Unlock()
		for _, cmd := range res.Commands {
			if err := r.grpcOut.HandleAndReportCommand(ctx, cmd); err != nil {
				log.Printf("[sim-agent-%03d] command failed: %v", r.id, err)
			}
		}
	}

	// Send metrics
	agent.GRPCSend(ctx, r.grpcOut, collected)
	r.sendCount.Add(1)
}
