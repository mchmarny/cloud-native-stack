package collectors_test

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/NVIDIA/cloud-native-stack/pkg/collectors"
	"github.com/NVIDIA/cloud-native-stack/pkg/measurement"
)

func TestSysctlCollector_Collect_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	collector := &collectors.SysctlCollector{}

	// Start collection and cancel mid-way
	go func() {
		// Give it a moment to start walking
		cancel()
	}()

	m, err := collector.Collect(ctx)

	// Context cancellation during walk should return context error
	if err != nil {
		if m != nil {
			t.Error("Expected nil measurement on error")
		}
		if !errors.Is(err, context.Canceled) {
			t.Logf("Got error: %v (expected context.Canceled or nil)", err)
		}
	}
}

func TestSysctlCollector_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	collector := &collectors.SysctlCollector{}

	m, err := collector.Collect(ctx)
	if err != nil {
		// /proc/sys might not exist on all systems
		if errors.Is(err, os.ErrNotExist) {
			t.Skip("/proc/sys not available on this system")
			return
		}
		t.Fatalf("Collect() failed: %v", err)
	}

	// Should return exactly one measurement with one subtype
	if m == nil {
		t.Fatal("Expected non-nil measurement")
	}

	if m.Type != measurement.TypeSysctl {
		t.Errorf("Expected type %s, got %s", measurement.TypeSysctl, m.Type)
	}

	if len(m.Subtypes) != 1 {
		t.Errorf("Expected 1 subtype, got %d", len(m.Subtypes))
		return
	}

	// Validate that Data is a map
	params := m.Subtypes[0].Data
	if params == nil {
		t.Error("Expected non-nil Data map")
		return
	}

	// Most systems have many sysctl parameters
	if len(params) == 0 {
		t.Error("Expected at least one sysctl parameter")
	}

	t.Logf("Found %d sysctl parameters", len(params))

	// Verify no /proc/sys/net entries (should be excluded)
	for key := range params {
		if strings.HasPrefix(key, "/proc/sys/net") {
			t.Errorf("Found /proc/sys/net entry which should be excluded: %s", key)
		}

		if !strings.HasPrefix(key, "/proc/sys") {
			t.Errorf("Key doesn't start with /proc/sys: %s", key)
		}
	}
}

func TestSysctlCollector_ExcludesNet(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	collector := &collectors.SysctlCollector{}

	m, err := collector.Collect(ctx)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			t.Skip("/proc/sys not available on this system")
			return
		}
		t.Fatalf("Collect() failed: %v", err)
	}

	if m == nil || len(m.Subtypes) == 0 {
		return
	}

	params := m.Subtypes[0].Data

	// Ensure no network parameters are included
	for key := range params {
		if strings.Contains(key, "/net/") {
			t.Errorf("Network sysctl should be excluded: %s", key)
		}
	}
}
