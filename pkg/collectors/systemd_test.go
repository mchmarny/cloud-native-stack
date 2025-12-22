package collectors_test

import (
	"context"
	"errors"
	"testing"

	"github.com/NVIDIA/cloud-native-stack/pkg/collectors"
	"github.com/NVIDIA/cloud-native-stack/pkg/measurement"
)

func TestSystemDCollector_Collect_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	collector := &collectors.SystemDCollector{
		Services: []string{"containerd.service"},
	}
	m, err := collector.Collect(ctx)

	// Should fail with context canceled
	if err != nil {
		if m != nil {
			t.Error("Expected nil measurement on error")
		}
		if !errors.Is(err, context.Canceled) {
			// D-Bus connection might fail for other reasons
			t.Logf("Got error: %v", err)
		}
	}
}

func TestSystemDCollector_DefaultServices(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()

	// Test with nil services (should use default)
	collector := &collectors.SystemDCollector{}

	m, err := collector.Collect(ctx)
	if err != nil {
		if m != nil {
			t.Error("Expected nil measurement on error")
		}
		// D-Bus might not be available or service might not exist
		t.Logf("Expected possible error for systemd access: %v", err)
		return
	}
}

func TestSystemDCollector_CustomServices(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()

	collector := &collectors.SystemDCollector{
		Services: []string{"containerd.service", "docker.service"},
	}

	m, err := collector.Collect(ctx)
	if err != nil {
		// Services might not exist or D-Bus unavailable
		t.Logf("Expected possible error: %v", err)
		return
	}

	// If successful, verify structure
	if m == nil {
		t.Fatal("Expected non-nil measurement")
	}

	if m.Type != measurement.TypeSystemD {
		t.Errorf("Expected type %s, got %s", measurement.TypeSystemD, m.Type)
	}

	// Should have subtypes for the services we requested
	if len(m.Subtypes) == 0 {
		t.Error("Expected at least one subtype")
	}

	for _, subtype := range m.Subtypes {
		if subtype.Name == "" {
			t.Error("Expected non-empty subtype name (service name)")
		}

		if subtype.Data == nil {
			t.Error("Expected non-nil Data map")
		}
	}
}

func TestSystemDCollector_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// This test requires systemd to be running
	ctx := context.Background()
	collector := &collectors.SystemDCollector{
		Services: []string{"containerd.service"},
	}

	m, err := collector.Collect(ctx)
	if err != nil {
		// SystemD might not be available on this system
		t.Skipf("SystemD not available or service not found: %v", err)
	}

	if m != nil && len(m.Subtypes) > 0 {
		t.Logf("Successfully collected %d systemd service configurations", len(m.Subtypes))
	}
}
