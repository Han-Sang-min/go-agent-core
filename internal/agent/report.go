package agent

import (
	"context"
	"fmt"
	"log"
	"math"
	"os"
	"sync/atomic"
	"time"

	"go-agent/internal/transport"
	pb "go-agent/proto/agentv1"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var (
	loc = mustLoadLocation("Asia/Seoul")
)

var counter atomic.Int64

type Collected struct {
	Seq int64
	TS  time.Time

	CPU  CPUStats
	Mem  MemStats
	Disk DiskStats
	Proc ProcStats

	K8s KubernetesMeta
}

func mustLoadLocation(src string) *time.Location {
	l, err := time.LoadLocation(src)
	if err != nil {
		panic(err)
	}
	return l
}

func formatBytes(b uint64) string {
	const (
		KB = 1024
		MB = 1024 * KB
		GB = 1024 * MB
	)

	switch {
	case b >= GB:
		return fmt.Sprintf("%.2fGiB", float64(b)/float64(GB))
	case b >= MB:
		return fmt.Sprintf("%.2fMiB", float64(b)/float64(MB))
	case b >= KB:
		return fmt.Sprintf("%.2fKiB", float64(b)/float64(KB))
	default:
		return fmt.Sprintf("%dB", b)
	}
}

type GRPCOut struct {
	cli *transport.Client
}

func NewGRPCOut(ctx context.Context, addr string) (*GRPCOut, error) {
	cli, err := transport.New(transport.Options{Addr: addr})
	if err != nil {
		return nil, err
	}
	hostname, _ := os.Hostname()
	if err := cli.Register(ctx, &pb.RegisterRequest{Hostname: hostname}); err != nil {
		_ = cli.Close()
		return nil, fmt.Errorf("register failed: %w", err)
	}

	return &GRPCOut{cli: cli}, nil
}

func (o *GRPCOut) Close() error {
	if o == nil || o.cli == nil {
		return nil
	}
	return o.cli.Close()
}

func (o *GRPCOut) AgentID() string {
	if o == nil || o.cli == nil || o.cli.Id == nil {
		return ""
	}
	return o.cli.Id.AgentId
}

func (o *GRPCOut) SendHeartbeat(ctx context.Context) (*pb.HeartbeatResponse, error) {
	if o.cli == nil {
		return nil, nil
	}
	agentID := o.AgentID()
	hostname, _ := os.Hostname()

	hb := &pb.Heartbeat{
		AgentId:  agentID,
		Hostname: hostname,
		Time:     timestamppb.Now(),
	}

	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	return o.cli.SendHeartbeat(ctx, hb)
}

func (o *GRPCOut) HandleAndReportCommand(ctx context.Context, cmd *pb.Command) error {
	if o.cli == nil {
		return nil
	}
	res := o.handleCommand(cmd)
	return o.ReportCommandResult(ctx, cmd, res)
}

func (o *GRPCOut) ReportCommandResult(ctx context.Context, cmd *pb.Command, out CommandOutcome) error {
	if o == nil || o.cli == nil {
		return nil
	}

	res := &pb.CommandResult{
		AgentId:   o.AgentID(),
		CommandId: "",
		Time:      timestamppb.Now(),
		Status:    out.Status,
		Output:    out.Output,
		Error:     out.Error,
	}

	if cmd != nil {
		res.CommandId = cmd.GetCommandId()
	}

	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	return o.cli.ReportCommandResult(ctx, res)
}

type CommandOutcome struct {
	Status pb.CommandResult_Status
	Output string
	Error  string
}

func (o *GRPCOut) handleCommand(cmd *pb.Command) CommandOutcome {
	switch cmd.GetName() {
	case "ping":
		return CommandOutcome{Status: pb.CommandResult_OK, Output: "pong"}
	case "snapshot":
		return CommandOutcome{Status: pb.CommandResult_OK, Output: "snapshot triggered"}
	default:
		return CommandOutcome{Status: pb.CommandResult_ERROR, Error: "unknown command"}
	}
}

type MetricPoint struct {
	Name  string
	Value float64
	Unit  string
}

func (o *GRPCOut) SendMetrics(ctx context.Context, metrics []MetricPoint) error {
	if o.cli == nil {
		return nil
	}
	agentID := o.AgentID()
	pbMetrics := make([]*pb.Metric, 0, len(metrics))

	for _, m := range metrics {
		pbMetrics = append(pbMetrics, &pb.Metric{
			Name:  m.Name,
			Value: m.Value,
			Unit:  m.Unit,
		})
	}

	mb := &pb.MetricBatch{
		AgentId: agentID,
		Time:    timestamppb.Now(),
		Metrics: pbMetrics,
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	return o.cli.SendMetrics(ctx, mb)
}

func Collect(ctx context.Context, env RuntimeEnv) Collected {
	seq := counter.Add(1)
	ts := time.Now().In(loc)

	var out Collected
	out.Seq, out.TS = seq, ts

	if cpu, err := env.CPU(ctx); err != nil {
		fmt.Printf("CPU Error: %v\n", err)
	} else {
		out.CPU = cpu
	}

	if mem, err := env.Mem(ctx); err != nil {
		fmt.Printf("Mem Error: %v\n", err)
	} else {
		out.Mem = mem
	}

	if disk, err := env.Disk(ctx); err != nil {
		fmt.Printf("Disk Error: %v\n", err)
	} else {
		out.Disk = disk
	}

	if proc, err := env.Procs(ctx); err != nil {
		fmt.Printf("Proc Error: %v\n", err)
	} else {
		out.Proc = proc
	}

	if kp, ok := env.(K8sMetaProvider); ok {
		meta, err := kp.K8sMeta(ctx)
		if err == nil {
			out.K8s = meta
		}
	}

	return out
}

func ToMetricPoints(c Collected) []MetricPoint {
	metrics := make([]MetricPoint, 0, 8)

	if c.CPU.Valid {
		metrics = append(metrics, MetricPoint{Name: "cpu.usage", Value: c.CPU.UsagePercent, Unit: "%"})
	}

	if c.Mem.Valid {
		if !math.IsNaN(c.Mem.UsedPercent) {
			metrics = append(metrics, MetricPoint{Name: "mem.used_percent", Value: c.Mem.UsedPercent, Unit: "%"})
		}
		metrics = append(metrics, MetricPoint{Name: "mem.used_bytes", Value: float64(c.Mem.UsedBytes), Unit: "bytes"})
	}

	if c.Disk.Valid {
		metrics = append(metrics, MetricPoint{Name: "disk.used_percent", Value: c.Disk.UsedPercent, Unit: "%"})
	}

	if c.Proc.Valid {
		metrics = append(metrics, MetricPoint{Name: "proc.count", Value: float64(c.Proc.Count), Unit: "count"})
	}

	return metrics
}

func GRPCSend(ctx context.Context, out *GRPCOut, c Collected) {
	metrics := ToMetricPoints(c)
	if err := out.SendMetrics(ctx, metrics); err != nil {
		log.Printf("[metrics] send failed: %v", err)
	}
}

func ConsoleOut(ctx context.Context, env RuntimeEnv, c Collected) {
	cpuStr := "    N/A"
	if c.CPU.Valid {
		cpuStr = fmt.Sprintf("%7.2f%%", c.CPU.UsagePercent)
	}

	memStr := "    N/A"
	if c.Mem.Valid {
		if math.IsNaN(c.Mem.UsedPercent) {
			memStr = fmt.Sprintf(" N/A (%s)", formatBytes(c.Mem.UsedBytes))
		} else {
			memStr = fmt.Sprintf("%7.2f%%", c.Mem.UsedPercent)
		}
	}

	diskStr := "    N/A"
	if c.Disk.Valid {
		diskStr = fmt.Sprintf("%7.2f%%", c.Disk.UsedPercent)
	}

	procStr := "    N/A"
	if c.Proc.Valid {
		procStr = fmt.Sprintf("%6d", c.Proc.Count)
	}

	ts := c.TS.Format("2006-01-02 15:04:05.000 MST")

	fmt.Printf(
		"[Seq:%6d] [Time:%s] CPU:%8s  Mem:%-10s  Disk:%7s  Procs:%6s\n",
		c.Seq, ts, cpuStr, memStr, diskStr, procStr,
	)
}
