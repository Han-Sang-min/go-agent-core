package simulator

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"
)

// Dashboard periodically prints a table of all agent states.
type Dashboard struct {
	agents []*AgentRunner

	mu    sync.RWMutex
	phase string
}

func NewDashboard(agents []*AgentRunner) *Dashboard {
	return &Dashboard{agents: agents}
}

func (d *Dashboard) SetPhase(name string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.phase = name
}

func (d *Dashboard) Run(ctx context.Context) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			d.render()
		}
	}
}

func (d *Dashboard) render() {
	d.mu.RLock()
	phase := d.phase
	d.mu.RUnlock()

	var sb strings.Builder

	sb.WriteString(fmt.Sprintf(
		"\n=== Simulator Dashboard [%s] Phase: %s ===\n",
		time.Now().Format("15:04:05"), phase,
	))
	sb.WriteString(fmt.Sprintf(
		"%-6s %-12s %-8s %-8s %-10s %-10s %-10s %-8s %-8s %-6s\n",
		"ID", "AgentUUID", "Conn", "NetDown", "CPU%", "Mem%", "Disk%", "Procs", "Sends", "Errs",
	))
	sb.WriteString(strings.Repeat("-", 96))
	sb.WriteString("\n")

	for _, a := range d.agents {
		st := a.Status()

		connStr := "NO"
		if st.Connected {
			connStr = "YES"
		}
		netStr := ""
		if st.NetworkDown {
			netStr = "DOWN"
		}

		cpuStr := "N/A"
		if st.LastMetrics.CPU.Valid {
			cpuStr = fmt.Sprintf("%.1f%%", st.LastMetrics.CPU.UsagePercent)
		}
		memStr := "N/A"
		if st.LastMetrics.Mem.Valid {
			memStr = fmt.Sprintf("%.1f%%", st.LastMetrics.Mem.UsedPercent)
		}
		diskStr := "N/A"
		if st.LastMetrics.Disk.Valid {
			diskStr = fmt.Sprintf("%.1f%%", st.LastMetrics.Disk.UsedPercent)
		}
		procStr := "N/A"
		if st.LastMetrics.Proc.Valid {
			procStr = fmt.Sprintf("%d", st.LastMetrics.Proc.Count)
		}

		uuid := st.AgentUUID
		if len(uuid) > 8 {
			uuid = uuid[:8]
		}

		sb.WriteString(fmt.Sprintf(
			"%-6d %-12s %-8s %-8s %-10s %-10s %-10s %-8s %-8d %-6d\n",
			st.ID, uuid, connStr, netStr,
			cpuStr, memStr, diskStr, procStr,
			st.SendCount, st.ErrorCount,
		))
	}

	if errMsg := d.findLastError(); errMsg != "" {
		sb.WriteString(fmt.Sprintf("\nLast error: %s\n", errMsg))
	}

	fmt.Print(sb.String())
}

func (d *Dashboard) findLastError() string {
	for _, a := range d.agents {
		st := a.Status()
		if st.LastError != "" {
			return fmt.Sprintf("agent-%03d: %s", st.ID, st.LastError)
		}
	}
	return ""
}

func (d *Dashboard) PrintSummary() {
	fmt.Println("\n========== SIMULATION SUMMARY ==========")
	var totalSends, totalErrors int64
	for _, a := range d.agents {
		st := a.Status()
		totalSends += st.SendCount
		totalErrors += st.ErrorCount
		fmt.Printf("  Agent %03d: sends=%d errors=%d connected=%v uuid=%s\n",
			st.ID, st.SendCount, st.ErrorCount, st.Connected, st.AgentUUID)
	}
	fmt.Printf("\n  Total: %d agents, %d sends, %d errors\n",
		len(d.agents), totalSends, totalErrors)
	fmt.Println("=========================================")
}
