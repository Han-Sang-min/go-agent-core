package collector

import (
	"fmt"
	"net"
	"time"

	pb "go-agent/proto/agentv1"
	"google.golang.org/grpc"
)

type grpcServer struct {
	addr string
	lis  net.Listener
	s    *grpc.Server
}

func newGRPCServer(cfg Config) (*grpcServer, error) {
	lis, err := net.Listen("tcp", cfg.ListenAddr)
	if err != nil {
		return nil, fmt.Errorf("listen %s: %w", cfg.ListenAddr, err)
	}

	s := grpc.NewServer()
	h := NewHandler()
	h.Init()

	pb.RegisterCollectorServiceServer(s, h)

	return &grpcServer{
		addr: cfg.ListenAddr,
		lis:  lis,
		s:    s,
	}, nil
}

func (g *grpcServer) Serve() error {
	return g.s.Serve(g.lis)
}

func (g *grpcServer) GracefulStop() {
	done := make(chan struct{})
	go func() {
		g.s.GracefulStop()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		g.s.Stop()
	}
}
