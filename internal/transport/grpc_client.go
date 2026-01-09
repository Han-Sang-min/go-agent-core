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

	id string
}

type Options struct {
	Addr string
}

func New(ctx context.Context, opt Options) (*Client, error) {
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

func (c *Client) Register(ctx context.Context, req *pb.RegisterRequest) (*pb.RegisterResponse, error) {
	c.id, err := c.api.Register(ctx, req)
	return err
}

func (c *Client) SendHeartbeat(ctx context.Context, hb *pb.Heartbeat) error {
	_, err := c.api.SendHeartbeat(ctx, hb)
	return err
}

func (c *Client) SendMetrics(ctx context.Context, mb *pb.MetricBatch) error {
	_, err := c.api.SendMetrics(ctx, mb)
	return err
}
