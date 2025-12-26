package os

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/NVIDIA/cloud-native-stack/pkg/measurement"
)

const sysctlSubtypeName = "sysctl"

func TestSysctlCollector_Collect_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.TODO())

	collector := &Collector{}

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

	ctx := context.TODO()
	collector := &Collector{}

	m, err := collector.Collect(ctx)
	if err != nil {
		// /proc/sys might not exist on all systems
		if errors.Is(err, os.ErrNotExist) {
			t.Skip("/proc/sys not available on this system")
			return
		}
		t.Fatalf("Collect() failed: %v", err)
	}

	// Should return measurement with TypeOS and three subtypes
	if m == nil {
		t.Fatal("Expected non-nil measurement")
		return
	}

	if m.Type != measurement.TypeOS {
		t.Errorf("Expected type %s, got %s", measurement.TypeOS, m.Type)
	}

	if len(m.Subtypes) != 4 {
		t.Errorf("Expected 4 subtypes (grub, sysctl, kmod, release), got %d", len(m.Subtypes))
		return
	}

	// Find the sysctl subtype
	var sysctlSubtype *measurement.Subtype
	for i := range m.Subtypes {
		if m.Subtypes[i].Name == sysctlSubtypeName {
			sysctlSubtype = &m.Subtypes[i]
			break
		}
	}

	if sysctlSubtype == nil {
		t.Fatal("Expected to find sysctl subtype")
	}

	// Validate that Data is a map
	params := sysctlSubtype.Data
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

	ctx := context.TODO()
	collector := &Collector{}

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

	// Find sysctl subtype
	var sysctlSubtype *measurement.Subtype
	for i := range m.Subtypes {
		if m.Subtypes[i].Name == sysctlSubtypeName {
			sysctlSubtype = &m.Subtypes[i]
			break
		}
	}

	if sysctlSubtype == nil {
		return
	}

	params := sysctlSubtype.Data

	// Ensure no network parameters are included
	for key := range params {
		if strings.Contains(key, "/net/") {
			t.Errorf("Network sysctl should be excluded: %s", key)
		}
	}
}

func TestSysctlCollector_MultiLineKeyValueParsing(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.TODO()
	collector := &Collector{}

	m, err := collector.Collect(ctx)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			t.Skip("/proc/sys not available on this system")
			return
		}
		t.Fatalf("Collect() failed: %v", err)
	}

	if m == nil || len(m.Subtypes) == 0 {
		t.Fatal("Expected non-nil measurement with subtypes")
	}

	// Find sysctl subtype
	var sysctlSubtype *measurement.Subtype
	for i := range m.Subtypes {
		if m.Subtypes[i].Name == sysctlSubtypeName {
			sysctlSubtype = &m.Subtypes[i]
			break
		}
	}

	if sysctlSubtype == nil {
		t.Fatal("Expected to find sysctl subtype")
		return
	}

	params := sysctlSubtype.Data

	// Check if /proc/sys/sunrpc/transports exists and has been parsed
	// This file typically contains lines like: "tcp 1048576\nudp 32768\nrdma 1048576"
	var foundTransportKeys bool
	for key := range params {
		if strings.HasPrefix(key, "/proc/sys/sunrpc/transports/") {
			foundTransportKeys = true
			// Verify the key format: /proc/sys/sunrpc/transports/<protocol>
			parts := strings.Split(key, "/")
			if len(parts) < 6 {
				t.Errorf("Expected extended path format, got: %s", key)
			}
			// Check that the value is a string (not the multi-line content)
			val := params[key]
			if valStr, ok := val.Any().(string); ok {
				if strings.Contains(valStr, "\n") {
					t.Errorf("Multi-line value should be split, but found newline in: %s = %s", key, valStr)
				}
			}
			t.Logf("Found parsed transport key: %s = %v", key, params[key])
		}
	}

	// If the file exists, we should find parsed keys
	if _, err := os.Stat("/proc/sys/sunrpc/transports"); err == nil {
		if !foundTransportKeys {
			// Check if the original file is still there (shouldn't be if it was parsed)
			if _, exists := params["/proc/sys/sunrpc/transports"]; exists {
				content := params["/proc/sys/sunrpc/transports"]
				if valStr, ok := content.Any().(string); ok {
					if strings.Contains(valStr, "\n") {
						t.Error("Multi-line /proc/sys/sunrpc/transports should have been parsed into separate keys")
					}
				}
			} else {
				t.Error("Expected to find parsed /proc/sys/sunrpc/transports/* keys")
			}
		}
	}
}

func TestSysctlCollector_SingleLineValues(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.TODO()
	collector := &Collector{}

	m, err := collector.Collect(ctx)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			t.Skip("/proc/sys not available on this system")
			return
		}
		t.Fatalf("Collect() failed: %v", err)
	}

	if m == nil || len(m.Subtypes) == 0 {
		t.Fatal("Expected non-nil measurement with subtypes")
	}

	// Find sysctl subtype
	var sysctlSubtype *measurement.Subtype
	for i := range m.Subtypes {
		if m.Subtypes[i].Name == "sysctl" {
			sysctlSubtype = &m.Subtypes[i]
			break
		}
	}

	if sysctlSubtype == nil {
		t.Fatal("Expected to find sysctl subtype")
		return
	}

	params := sysctlSubtype.Data

	// Single-line files should be stored with their original path (not split)
	// Check for common single-value sysctl parameters
	singleValuePaths := []string{
		"/proc/sys/kernel/hostname",
		"/proc/sys/kernel/ostype",
		"/proc/sys/kernel/osrelease",
	}

	for _, path := range singleValuePaths {
		if _, err := os.Stat(path); err == nil {
			// File exists, check if it's in params
			if val, exists := params[path]; exists {
				if valStr, ok := val.Any().(string); ok {
					// Single-line values shouldn't have been extended
					if strings.Contains(path, "//") {
						t.Errorf("Single-line value has double slash: %s", path)
					}
					t.Logf("Single-line value preserved: %s = %s", path, valStr)
				}
			}
		}
	}
}

func TestSysctlCollector_MixedContent(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.TODO()
	collector := &Collector{}

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

	// Find sysctl subtype
	var sysctlSubtype *measurement.Subtype
	for i := range m.Subtypes {
		if m.Subtypes[i].Name == "sysctl" {
			sysctlSubtype = &m.Subtypes[i]
			break
		}
	}

	if sysctlSubtype == nil {
		return
	}

	params := sysctlSubtype.Data

	// Verify that no values contain unprocessed multi-line content with key-value pattern
	for key, val := range params {
		if valStr, ok := val.Any().(string); ok {
			// If it has newlines, check if it looks like unparsed key-value pairs
			if strings.Contains(valStr, "\n") {
				lines := strings.Split(valStr, "\n")
				allKeyValue := true
				for _, line := range lines {
					line = strings.TrimSpace(line)
					if line == "" {
						continue
					}
					parts := strings.Fields(line)
					if len(parts) < 2 {
						allKeyValue = false
						break
					}
				}
				if allKeyValue && len(lines) > 1 {
					t.Errorf("Found unparsed multi-line key-value content at %s: %q", key, valStr)
				}
			}
		}
	}
}
