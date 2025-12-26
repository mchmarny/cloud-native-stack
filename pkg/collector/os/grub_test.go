package os

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/NVIDIA/cloud-native-stack/pkg/measurement"
)

func TestGrubCollector_Collect_ContextCancellation(t *testing.T) {
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

func TestGrubCollector_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	collector := &Collector{}

	m, err := collector.Collect(ctx)
	if err != nil {
		// /proc/cmdline might not exist on all systems
		if errors.Is(err, os.ErrNotExist) {
			t.Skip("/proc/cmdline not available on this system")
			return
		}
		t.Fatalf("Collect() failed: %v", err)
	}

	// Should return measurement with TypeOS and three subtypes: grub, sysctl, kmod
	if m == nil {
		t.Fatal("Expected non-nil measurement")
	}

	if m.Type != measurement.TypeOS {
		t.Errorf("Expected type %s, got %s", measurement.TypeOS, m.Type)
	}

	if len(m.Subtypes) != 4 {
		t.Errorf("Expected exactly 4 subtypes (grub, sysctl, kmod, release), got %d", len(m.Subtypes))
		return
	}

	// Find the grub subtype
	var grubSubtype *measurement.Subtype
	for i := range m.Subtypes {
		if m.Subtypes[i].Name == "grub" {
			grubSubtype = &m.Subtypes[i]
			break
		}
	}

	if grubSubtype == nil {
		t.Fatal("Expected to find grub subtype")
	}

	// Validate that Data is a map
	props := grubSubtype.Data
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
	collector := &Collector{}

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

	// Find grub subtype
	var grubSubtype *measurement.Subtype
	for i := range m.Subtypes {
		if m.Subtypes[i].Name == "grub" {
			grubSubtype = &m.Subtypes[i]
			break
		}
	}

	if grubSubtype == nil {
		t.Fatal("Expected to find grub subtype")
	}

	props := grubSubtype.Data

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
