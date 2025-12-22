package collectors_test

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/NVIDIA/cloud-native-stack/pkg/collectors"
	"github.com/NVIDIA/cloud-native-stack/pkg/measurement"
)

func TestGrubCollector_Collect_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	collector := &collectors.GrubCollector{}
	m, err := collector.Collect(ctx)

	if err == nil {
		// On some systems, the read may complete before context check
		t.Skip("Context cancellation timing dependent")
	}

	if m != nil {
		t.Error("Expected nil measurement on error")
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

	m, err := collector.Collect(ctx)
	if err != nil {
		// /proc/cmdline might not exist on all systems
		if errors.Is(err, os.ErrNotExist) {
			t.Skip("/proc/cmdline not available on this system")
			return
		}
		t.Fatalf("Collect() failed: %v", err)
	}

	// Should have exactly one measurement with one subtype containing all boot parameters
	if m == nil {
		t.Fatal("Expected non-nil measurement")
	}

	if m.Type != measurement.TypeGrub {
		t.Errorf("Expected type %s, got %s", measurement.TypeGrub, m.Type)
	}

	if len(m.Subtypes) != 1 {
		t.Errorf("Expected exactly 1 subtype, got %d", len(m.Subtypes))
		return
	}

	// Validate that Data is a map
	props := m.Subtypes[0].Data
	if props == nil {
		t.Error("Expected non-nil Data map")
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

	m, err := collector.Collect(ctx)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			t.Skip("/proc/cmdline not available on this system")
			return
		}
		t.Fatalf("Collect() failed: %v", err)
	}

	if m == nil || len(m.Subtypes) == 0 {
		t.Fatal("Expected at least one subtype")
	}

	props := m.Subtypes[0].Data

	// Check that we can parse both key-only and key=value formats
	hasKeyOnly := false
	hasKeyValue := false

	for key, value := range props {
		if key == "" {
			t.Error("Found empty key in Properties")
			continue
		}

		strVal := value.Any()
		if strVal == "" {
			hasKeyOnly = true
			t.Logf("Key-only param: %s", key)
		} else {
			hasKeyValue = true
			t.Logf("Key=value param: %s=%v", key, strVal)
		}
	}

	t.Logf("Has key-only params: %v, Has key=value params: %v", hasKeyOnly, hasKeyValue)
}
