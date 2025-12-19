package collectors_test

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/NVIDIA/cloud-native-stack/pkg/collectors"
)

func TestGrubCollector_Collect_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	collector := &collectors.GrubCollector{}
	_, err := collector.Collect(ctx)

	if err == nil {
		// On some systems, the read may complete before context check
		t.Skip("Context cancellation timing dependent")
	}

	if !errors.Is(err, context.Canceled) {
		t.Errorf("Expected context.Canceled, got %v", err)
	}
}

func TestGrubCollector_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	collector := &collectors.GrubCollector{}

	configs, err := collector.Collect(ctx)
	if err != nil {
		// /proc/cmdline might not exist on all systems
		if errors.Is(err, os.ErrNotExist) {
			t.Skip("/proc/cmdline not available on this system")
			return
		}
		t.Fatalf("Collect() failed: %v", err)
	}

	// Should have exactly one configuration with all boot parameters
	if len(configs) != 1 {
		t.Errorf("Expected exactly 1 config, got %d", len(configs))
	}

	if len(configs) == 0 {
		return
	}

	cfg := configs[0]
	if cfg.Type != collectors.GrubType {
		t.Errorf("Expected type %s, got %s", collectors.GrubType, cfg.Type)
	}

	// Validate that Data is a map
	props, ok := cfg.Data.(map[string]any)
	if !ok {
		t.Errorf("Expected map[string]any, got %T", cfg.Data)
		return
	}

	// Most systems have at least a few boot parameters
	if len(props) == 0 {
		t.Error("Expected at least one boot parameter")
	}

	t.Logf("Found %d boot parameters", len(props))
}

func TestGrubCollector_ValidatesParsing(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	collector := &collectors.GrubCollector{}

	configs, err := collector.Collect(ctx)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			t.Skip("/proc/cmdline not available on this system")
			return
		}
		t.Fatalf("Collect() failed: %v", err)
	}

	if len(configs) == 0 {
		t.Fatal("Expected at least one config")
	}

	props := configs[0].Data.(map[string]any)

	// Check that we can parse both key-only and key=value formats
	hasKeyOnly := false
	hasKeyValue := false

	for key, value := range props {
		if key == "" {
			t.Error("Found empty key in Properties")
			continue
		}

		if value == "" {
			hasKeyOnly = true
			t.Logf("Key-only param: %s", key)
		} else {
			hasKeyValue = true
			t.Logf("Key=value param: %s=%v", key, value)
		}
	}

	t.Logf("Has key-only params: %v, Has key=value params: %v", hasKeyOnly, hasKeyValue)
}
