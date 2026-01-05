package collector

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestHostEnv_Integration(t *testing.T) {
	tmpRoot := t.TempDir()

	procDir := filepath.Join(tmpRoot, "proc")
	if err := os.MkdirAll(procDir, 0755); err != nil {
		t.Fatal(err)
	}

	meminfoContent := `MemTotal:        16000000 kB
MemAvailable:     8000000 kB
`
	if err := os.WriteFile(filepath.Join(procDir, "meminfo"), []byte(meminfoContent), 0644); err != nil {
		t.Fatal(err)
	}

	statContent := "cpu  1000 0 500 2000 100 10 20 0\n"
	if err := os.WriteFile(filepath.Join(procDir, "stat"), []byte(statContent), 0644); err != nil {
		t.Fatal(err)
	}

	env := NewHostEnv(tmpRoot)

	ctx := context.Background()
	mem, err := env.Mem(ctx)
	if err != nil {
		t.Errorf("Mem() error: %v", err)
	}

	if mem.UsedPercent != 50.0 {
		t.Errorf("Expected 50%% usage, got %f%%", mem.UsedPercent)
	}
}
