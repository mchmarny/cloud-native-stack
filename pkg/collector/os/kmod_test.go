package os

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/NVIDIA/cloud-native-stack/pkg/measurement"
)

func TestKModCollector_Collect(t *testing.T) {
	ctx := context.Background()
	collector := &Collector{}

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

	collector := &Collector{}
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
	collector := &Collector{}

	m, err := collector.Collect(ctx)
	if err != nil {
		// /proc/modules might not exist on all systems
		if errors.Is(err, os.ErrNotExist) {
			t.Skip("/proc/modules not available")
			return
		}
		t.Fatalf("Collect() failed: %v", err)
	}

	// Should return measurement with TypeOS and three subtypes
	if m == nil {
		t.Fatal("Expected non-nil measurement")
	}

	if m.Type != measurement.TypeOS {
		t.Errorf("Expected type %s, got %s", measurement.TypeOS, m.Type)
	}

	if len(m.Subtypes) != 4 {
		t.Errorf("Expected 4 subtypes (grub, sysctl, kmod, release), got %d", len(m.Subtypes))
		return
	}

	// Find the kmod subtype
	var kmodSubtype *measurement.Subtype
	for i := range m.Subtypes {
		if m.Subtypes[i].Name == "kmod" {
			kmodSubtype = &m.Subtypes[i]
			break
		}
	}

	if kmodSubtype == nil {
		t.Fatal("Expected to find kmod subtype")
	}

	// Validate that Data contains module names
	data := kmodSubtype.Data
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
