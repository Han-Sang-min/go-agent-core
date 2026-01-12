package transport

import (
	"context"
	"fmt"
	"time"

	pb "go-agent/proto/agentv1"

	"google.golang.org/grpc"
	"google.golang.org/grpc/backoff"
	"google.golang.org/grpc/credentials/insecure"
)

type Client struct {
	cc  *grpc.ClientConn
	api pb.CollectorServiceClient

	Id *pb.RegisterResponse
}

type Options struct {
	Addr string
}

func New(opt Options) (*Client, error) {
	if opt.Addr == "" {
		return nil, fmt.Errorf("empty grpc addr")
	}

	cc, err := grpc.NewClient(
		opt.Addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithConnectParams(grpc.ConnectParams{
			Backoff: backoff.Config{
				BaseDelay:  200 * time.Millisecond,
				Multiplier: 1.6,
				MaxDelay:   5 * time.Second,
			},
			MinConnectTimeout: 3 * time.Second,
		}),
	)
	if err != nil {
		return nil, err
	}

	return &Client{
		cc:  cc,
		api: pb.NewCollectorServiceClient(cc),
	}, nil
}

func (c *Client) Close() error {
	return c.cc.Close()
}

func (c *Client) Register(ctx context.Context, req *pb.RegisterRequest) error {
	id, err := c.api.Register(ctx, req)
	c.Id = id
	return err
}

func (c *Client) SendHeartbeat(ctx context.Context, hb *pb.Heartbeat) (*pb.HeartbeatResponse, error) {
	resp, err := c.api.SendHeartbeat(ctx, hb)
	return resp, err
}

func (c *Client) ReportCommandResult(ctx context.Context, res *pb.CommandResult) error {
	_, err := c.api.ReportCommandResult(ctx, res)
	return err
}

func (c *Client) SendMetrics(ctx context.Context, mb *pb.MetricBatch) error {
	_, err := c.api.SendMetrics(ctx, mb)
	return err
}
