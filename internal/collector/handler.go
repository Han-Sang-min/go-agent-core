package collector

import (
	"context"
	"encoding/json"
	"log"
	"os"
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

type AgentState struct {
	FirstSeen time.Time
	LastSeen  time.Time
	BootId    string

	NodeName string
	Hostname string

	Pending []*pb.Command
}

type agentSnapshot struct {
	AgentID   string    `json:"agent_id"`
	Hostname  string    `json:"hostname"`
	NodeName  string    `json:"node_name"`
	BootId    string    `json:"boot_id"`
	FirstSeen time.Time `json:"first_seen"`
	LastSeen  time.Time `json:"last_seen"`
}

type Handler struct {
	pb.UnimplementedCollectorServiceServer

	mu        sync.Mutex
	agents    map[string]*AgentState
	ttl       time.Duration
	statePath string
}

func mustLoadLocation(src string) *time.Location {
	l, err := time.LoadLocation(src)
	if err != nil {
		panic(err)
	}
	return l
}

func NewHandler(statePath string) *Handler {
	h := &Handler{
		agents:    make(map[string]*AgentState),
		ttl:       60 * time.Second,
		statePath: statePath,
	}
	h.loadState()
	go h.gcLoop(10 * time.Second)

	return h
}

func (h *Handler) saveState() {
	if h.statePath == "" {
		return
	}

	h.mu.Lock()
	snapshots := make([]agentSnapshot, 0, len(h.agents))
	for id, st := range h.agents {
		snapshots = append(snapshots, agentSnapshot{
			AgentID:   id,
			Hostname:  st.Hostname,
			NodeName:  st.NodeName,
			BootId:    st.BootId,
			FirstSeen: st.FirstSeen,
			LastSeen:  st.LastSeen,
		})
	}
	h.mu.Unlock()

	data, err := json.Marshal(snapshots)
	if err != nil {
		log.Printf("[state] marshal failed: %v", err)
		return
	}
	if err := os.WriteFile(h.statePath, data, 0644); err != nil {
		log.Printf("[state] write failed: %v", err)
	}
}

func (h *Handler) loadState() {
	if h.statePath == "" {
		return
	}

	data, err := os.ReadFile(h.statePath)
	if err != nil {
		if !os.IsNotExist(err) {
			log.Printf("[state] read failed: %v", err)
		}
		return
	}

	var snapshots []agentSnapshot
	if err := json.Unmarshal(data, &snapshots); err != nil {
		log.Printf("[state] unmarshal failed: %v", err)
		return
	}

	h.mu.Lock()
	for _, s := range snapshots {
		h.agents[s.AgentID] = &AgentState{
			FirstSeen: s.FirstSeen,
			LastSeen:  s.LastSeen,
			BootId:    s.BootId,
			NodeName:  s.NodeName,
			Hostname:  s.Hostname,
		}
	}
	h.mu.Unlock()

	log.Printf("[state] loaded %d agents", len(snapshots))
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
		changed := false
		for agentID, st := range h.agents {
			if now.Sub(st.LastSeen) > h.ttl {
				delete(h.agents, agentID)
				log.Printf("[gc] expired agent_id=%s", agentID)
				changed = true
			}
		}
		h.mu.Unlock()

		if changed {
			h.saveState()
		}
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
		BootId:    uuid.NewString(),
		NodeName:  req.Nodename,
		Hostname:  req.Hostname,
		Pending: []*pb.Command{
			{
				CommandId: "boot-" + agentID,
				Name:      "ping",
				ArgsJson:  `{}`,
			},
		},
	}
	h.mu.Unlock()

	log.Printf("[register] agent_id=%s node=%s host=%s", agentID, req.GetNodename(), req.GetHostname())
	h.saveState()
	return &pb.RegisterResponse{AgentId: agentID}, nil
}

func (h *Handler) ReportCommandResult(ctx context.Context, res *pb.CommandResult) (*pb.Ack, error) {
	if res.GetAgentId() == "" {
		return nil, status.Error(codes.InvalidArgument, "agent_id is required")
	}
	if res.GetCommandId() == "" {
		return nil, status.Error(codes.InvalidArgument, "command_id is required")
	}

	log.Printf("[cmd-result] agent_id=%s cmd_id=%s status=%s output=%q err=%q",
		res.GetAgentId(),
		res.GetCommandId(),
		res.GetStatus().String(),
		res.GetOutput(),
		res.GetError(),
	)

	return &pb.Ack{Ok: true, Message: "command result received"}, nil
}

func (h *Handler) SendHeartbeat(ctx context.Context, req *pb.Heartbeat) (*pb.HeartbeatResponse, error) {
	agentID := req.GetAgentId()
	if agentID == "" {
		return nil, status.Error(codes.InvalidArgument, "agent_id is required")
	}

	h.mu.Lock()
	st, ok := h.agents[agentID]
	if !ok {
		h.mu.Unlock()
		return nil, status.Error(codes.NotFound, "unknown agent_id")
	}

	st.LastSeen = time.Now()

	cmds := st.Pending
	st.Pending = nil

	h.mu.Unlock()

	log.Printf("[hb] agent_id=%s host=%s cmds=%d", agentID, req.GetHostname(), len(cmds))

	return &pb.HeartbeatResponse{
		Ok:       true,
		Commands: cmds,
	}, nil
}

func (h *Handler) SendMetrics(ctx context.Context, req *pb.MetricBatch) (*pb.Ack, error) {
	if req.GetAgentId() == "" {
		return nil, status.Error(codes.InvalidArgument, "agent_id is required")
	}

	h.mu.Lock()
	st := h.agents[req.GetAgentId()]
	h.mu.Unlock()

	nodeName := "unknown"
	if st != nil {
		nodeName = st.NodeName
	}

	for _, metric := range req.Metrics {
		log.Printf("[%s][%s / %s] %f labels=%v", metric.Name, req.AgentId, nodeName, metric.Value, req.GetLabels())
	}
	return &pb.Ack{Ok: true, Message: "metrics received"}, nil
}
