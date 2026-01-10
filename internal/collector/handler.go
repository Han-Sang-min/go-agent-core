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

var (
	loc = mustLoadLocation("Asia/Seoul")
)

// 커맨드는 어떤식으로 보내는가? 모았다가? 커맨드는 애초에 collector가 반사적으로 보내는건가?
type AgentState struct {
	FirstSeen time.Time
	LastSeen  time.Time
	BootId    string

	Pending []*pb.Command
}

type Handler struct {
	pb.UnimplementedCollectorServiceServer

	mu     sync.Mutex
	agents map[string]*AgentState
	ttl    time.Duration
}

func mustLoadLocation(src string) *time.Location {
	l, err := time.LoadLocation(src)
	if err != nil {
		panic(err)
	}
	return l
}

func NewHandler() *Handler {
	h := &Handler{
		agents: make(map[string]*AgentState),
		ttl:    60 * time.Second,
	}
	go h.gcLoop(10 * time.Second)

	return h
}

func (h *Handler) Init() {
	loc, err := time.LoadLocation("Asia/Seoul")
	if err != nil {
		panic(err)
	}
	time.Local = loc
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
	agentId := req.GetAgentId()
	if req.GetAgentId() == "" {
		return nil, status.Error(codes.InvalidArgument, "agent_id is required")
	}

	h.mu.Lock()
	st, ok := h.agents[agentId]
	if !ok {
		h.mu.Unlock()
		return nil, status.Error(codes.NotFound, "unknown agent_id")
	}
	st.LastSeen = time.Now()
	h.mu.Unlock()

	log.Printf("[hb] agent_id=%s host=%s", req.GetAgentId(), req.GetHostname())
	return &pb.Ack{Ok: true, Message: "heartbeat received"}, nil
}

func (h *Handler) SendMetrics(ctx context.Context, req *pb.MetricBatch) (*pb.Ack, error) {
	if req.GetAgentId() == "" {
		return nil, status.Error(codes.InvalidArgument, "agent_id is required")
	}
	for _, metric := range req.Metrics {
		log.Printf("[%s][%s] %f", metric.Name, req.AgentId, metric.Value)
	}
	return &pb.Ack{Ok: true, Message: "metrics received"}, nil
}
