package collector

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"

	pb "go-agent/proto/agentv1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type AgentState struct {
	FirstSeen time.Time
	LastSeen  time.Time
	BootId    string
}

type Handler struct {
	pb.UnimplementedCollectorServiceServer

	mu     sync.Mutex
	agents map[string]*AgentState
	ttl    time.Duration
}

func NewHandler() *Handler {
	h := &Handler{
		agents: make(map[string]*AgentState),
		ttl:    60 * time.Second,
	}
	go h.gcLoop(10 * time.Second)
	return h
}

func (h *Handler) gcLoop(interval time.Duration) {
	t := time.NewTicker(interval)
	defer t.Stop()

	for range t.C {
		now := time.Now()

		h.mu.Lock()
		for agentID, st := range h.agents {
			if now.Sub(st.LastSeen) > h.ttl {
				delete(h.agents, agentID)
				log.Printf("[gc] expired agent_id=%s", agentID)
			}
		}
		h.mu.Unlock()
	}
}

func (h *Handler) Register(ctx context.Context, req *pb.RegisterRequest) (*pb.RegisterResponse, error) {
	if req.GetHostname() == "" {
		return nil, status.Error(codes.InvalidArgument, "hostname is required")
	}

	now := time.Now()
	agentID := uuid.NewString()

	h.mu.Lock()
	h.agents[agentID] = &AgentState{
		FirstSeen: now,
		LastSeen:  now,
	}
	h.mu.Unlock()

	log.Printf("[register] agent_id=%s host=%s", agentID, req.GetHostname())

	return &pb.RegisterResponse{
		AgentId: agentID,
	}, nil
}

func (h *Handler) SendHeartbeat(ctx context.Context, req *pb.Heartbeat) (*pb.Ack, error) {
	if req.GetAgentId() == "" {
		return nil, status.Error(codes.InvalidArgument, "agent_id is required")
	}
	log.Printf("[hb] agent_id=%s host=%s", req.GetAgentId(), req.GetHostname())
	return &pb.Ack{Ok: true, Message: "heartbeat received"}, nil
}

func (h *Handler) SendMetrics(ctx context.Context, req *pb.MetricBatch) (*pb.Ack, error) {
	if req.GetAgentId() == "" {
		return nil, status.Error(codes.InvalidArgument, "agent_id is required")
	}
	log.Printf("[metrics] agent_id=%s metrics=%d", req.GetAgentId(), len(req.GetMetrics()))
	return &pb.Ack{Ok: true, Message: "metrics received"}, nil
}
