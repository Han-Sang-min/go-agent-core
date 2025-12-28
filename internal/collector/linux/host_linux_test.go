//go:build linux

package collector

import (
	"context"
	"testing"
	"time"
)

func TestReadCPU(t *testing.T) {
	ctx, canel := context.WithCancel(context.Background())
	defer canel()

	collecorCh := make(chan HostMeterics, 1)
	go Collector(ctx, collecorCh)

	// timeout := time.NewTimer(2 * time.Second)
	// defer timeout.Stop()

	for {
		select {
		case usage := <-collecorCh:
			t.Logf("CPU Usage: %.2f%%\n", usage.CpuPercent)
			t.Logf("MEM Usage: %.2f%%\n", usage.MemPercent)
			return
		case <-time.After(5 * time.Second):
			t.Fatal("no cpu usage received within 5s")
		}
	}
}
