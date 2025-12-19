package collectors_test

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/NVIDIA/cloud-native-stack/pkg/collectors"
)

func TestKModCollector_Collect(t *testing.T) {
	ctx := context.Background()
	collector := &collectors.KModCollector{}

	// This test validates the interface works correctly
	_, err := collector.Collect(ctx)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			t.Skip("/proc/modules not available on this system")
			return
		}
		if !errors.Is(err, os.ErrPermission) {
			t.Errorf("Collect() unexpected error = %v", err)
		}
	}
}

func TestKModCollector_Collect_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	collector := &collectors.KModCollector{}
	_, err := collector.Collect(ctx)

	if err == nil {
		// On some systems, the read may complete before context check
		t.Skip("Context cancellation timing dependent")
	}

	if !errors.Is(err, context.Canceled) {
		t.Errorf("Expected context.Canceled, got %v", err)
	}
}

func TestKModCollector_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	collector := &collectors.KModCollector{}

	configs, err := collector.Collect(ctx)
	if err != nil {
		// /proc/modules might not exist on all systems
		if errors.Is(err, os.ErrNotExist) {
			t.Skip("/proc/modules not available")
			return
		}
		t.Fatalf("Collect() failed: %v", err)
	}

	// Should return exactly one configuration
	if len(configs) != 1 {
		t.Errorf("Expected 1 config, got %d", len(configs))
	}

	if len(configs) == 0 {
		return
	}

	cfg := configs[0]
	if cfg.Type != collectors.KModType {
		t.Errorf("Expected type %s, got %s", collectors.KModType, cfg.Type)
	}

	// Validate that Data is a string slice
	modules, ok := cfg.Data.([]string)
	if !ok {
		t.Errorf("Expected []string, got %T", cfg.Data)
		return
	}

	// Most systems have at least a few kernel modules loaded
	if len(modules) == 0 {
		t.Error("Expected at least one kernel module")
	}

	t.Logf("Found %d kernel modules", len(modules))
}
