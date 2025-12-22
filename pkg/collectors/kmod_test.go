package collectors_test

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/NVIDIA/cloud-native-stack/pkg/collectors"
	"github.com/NVIDIA/cloud-native-stack/pkg/measurement"
)

func TestKModCollector_Collect(t *testing.T) {
	ctx := context.Background()
	collector := &collectors.KModCollector{}

	// This test validates the interface works correctly
	m, err := collector.Collect(ctx)
	if err != nil {
		if m != nil {
			t.Error("Expected nil measurement on error")
		}
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

func TestKModCollector_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	collector := &collectors.KModCollector{}

	m, err := collector.Collect(ctx)
	if err != nil {
		// /proc/modules might not exist on all systems
		if errors.Is(err, os.ErrNotExist) {
			t.Skip("/proc/modules not available")
			return
		}
		t.Fatalf("Collect() failed: %v", err)
	}

	// Should return exactly one measurement with one subtype
	if m == nil {
		t.Fatal("Expected non-nil measurement")
	}

	if m.Type != measurement.TypeKMod {
		t.Errorf("Expected type %s, got %s", measurement.TypeKMod, m.Type)
	}

	if len(m.Subtypes) != 1 {
		t.Errorf("Expected 1 subtype, got %d", len(m.Subtypes))
		return
	}

	// Validate that Data contains module names
	data := m.Subtypes[0].Data
	if data == nil {
		t.Error("Expected non-nil Data map")
		return
	}

	// Most systems have at least a few kernel modules loaded
	if len(data) == 0 {
		t.Error("Expected at least one kernel module")
	}

	t.Logf("Found %d loaded kernel modules", len(data))
}
