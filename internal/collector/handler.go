package collector

import (
	"context"
	"log"

	pb "go-agent/proto/agentv1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Handler struct {
	pb.UnimplementedCollectorServiceServer
}

func NewHandler() *Handler {
	return &Handler{}
}

func (h *Handler) SendHeartbeat(ctx context.Context, req *pb.Heartbeat) (*pb.Ack, error) {
	if req.GetAgentId() == "" {
		return nil, status.Error(codes.InvalidArgument, "agent_id is required")
	}
	log.Printf("[hb] agent_id=%s host=%s", req.GetAgentId(), req.GetHostname())
	return &pb.Ack{Ok: true, Message: "heartbeat received"}, nil
}

func (h *Handler) sendMetrics(ctx context.Context, req *pb.MetricBatch) (*pb.Ack, error) {
	if req.GetAgentId() == "" {
		return nil, status.Error(codes.InvalidArgument, "agent_id is required")
	}
	log.Printf("[metrics] agent_id=%s metrics=%d", req.GetAgentId(), len(req.GetMetrics()))
	return &pb.Ack{Ok: true, Message: "metrics received"}, nil
}
