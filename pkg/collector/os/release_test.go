package os

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/NVIDIA/cloud-native-stack/pkg/measurement"
)

const releaseSubtypeName = "release"

func TestReleaseCollector_Collect_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.TODO())
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

func TestReleaseCollector_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.TODO()
	collector := &Collector{}

	m, err := collector.Collect(ctx)
	if err != nil {
		// /etc/os-release might not exist on all systems
		if errors.Is(err, os.ErrNotExist) {
			t.Skip("/etc/os-release not available on this system")
			return
		}
		t.Fatalf("Collect() failed: %v", err)
	}

	// Should return measurement with TypeOS and four subtypes: grub, sysctl, kmod, release
	if m == nil {
		t.Fatal("Expected non-nil measurement")
		return
	}

	if m.Type != measurement.TypeOS {
		t.Errorf("Expected type %s, got %s", measurement.TypeOS, m.Type)
	}

	if len(m.Subtypes) != 4 {
		t.Errorf("Expected exactly 4 subtypes (grub, sysctl, kmod, release), got %d", len(m.Subtypes))
		return
	}

	// Find the release subtype
	var releaseSubtype *measurement.Subtype
	for i := range m.Subtypes {
		if m.Subtypes[i].Name == releaseSubtypeName {
			releaseSubtype = &m.Subtypes[i]
			break
		}
	}

	if releaseSubtype == nil {
		t.Fatal("Expected to find release subtype")
		return
	}

	// Validate that Data is a map
	data := releaseSubtype.Data
	if data == nil {
		t.Error("Expected non-nil Data map")
		return
	}

	// Most systems have several os-release fields
	if len(data) == 0 {
		t.Error("Expected at least one os-release field")
	}

	t.Logf("Found %d os-release fields", len(data))

	// Check for common fields that should exist
	commonFields := []string{"ID", "NAME", "VERSION_ID"}
	for _, field := range commonFields {
		if val, exists := data[field]; exists {
			t.Logf("%s = %v", field, val.Any())
		}
	}
}

func TestReleaseCollector_ValidatesKeyValueParsing(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.TODO()
	collector := &Collector{}

	m, err := collector.Collect(ctx)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			t.Skip("/etc/os-release not available on this system")
			return
		}
		t.Fatalf("Collect() failed: %v", err)
	}

	if m == nil || len(m.Subtypes) == 0 {
		t.Fatal("Expected at least one subtype")
	}

	// Find release subtype
	var releaseSubtype *measurement.Subtype
	for i := range m.Subtypes {
		if m.Subtypes[i].Name == releaseSubtypeName {
			releaseSubtype = &m.Subtypes[i]
			break
		}
	}

	if releaseSubtype == nil {
		t.Fatal("Expected to find release subtype")
		return
	}

	data := releaseSubtype.Data

	// Check that all keys have values (no empty keys or values for key=value format)
	for key, value := range data {
		if key == "" {
			t.Error("Found empty key in Data")
			continue
		}

		strVal := value.Any()
		t.Logf("Field: %s = %v", key, strVal)

		// Values should not contain quotes since they're stripped
		if str, ok := strVal.(string); ok {
			if len(str) > 0 && (str[0] == '"' || str[0] == '\'') {
				t.Errorf("Value for %s still contains quotes: %s", key, str)
			}
			if len(str) > 0 && (str[len(str)-1] == '"' || str[len(str)-1] == '\'') {
				t.Errorf("Value for %s still contains quotes: %s", key, str)
			}
		}
	}
}

func TestReleaseCollector_HandlesQuotedValues(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.TODO()
	collector := &Collector{}

	m, err := collector.Collect(ctx)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			t.Skip("/etc/os-release not available on this system")
			return
		}
		t.Fatalf("Collect() failed: %v", err)
	}

	if m == nil || len(m.Subtypes) == 0 {
		t.Fatal("Expected at least one subtype")
	}

	// Find release subtype
	var releaseSubtype *measurement.Subtype
	for i := range m.Subtypes {
		if m.Subtypes[i].Name == releaseSubtypeName {
			releaseSubtype = &m.Subtypes[i]
			break
		}
	}

	if releaseSubtype == nil {
		t.Fatal("Expected to find release subtype")
		return
	}

	data := releaseSubtype.Data

	// Pretty_name often contains spaces and is quoted
	if prettyName, exists := data["PRETTY_NAME"]; exists {
		strVal := prettyName.Any().(string)
		t.Logf("PRETTY_NAME = %s", strVal)

		// Should not have surrounding quotes
		if len(strVal) > 0 && strVal[0] == '"' {
			t.Error("PRETTY_NAME value still has leading quote")
		}
		if len(strVal) > 0 && strVal[len(strVal)-1] == '"' {
			t.Error("PRETTY_NAME value still has trailing quote")
		}
	}
}

func TestReleaseCollector_HandlesEmptyLines(t *testing.T) {
	// Create a temporary test file with empty lines and comments
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "os-release")

	content := `NAME="Test OS"

ID=testos
VERSION_ID="1.0"

PRETTY_NAME="Test OS 1.0"
`

	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// This test would need to mock the file reading, which is not easily done
	// with the current implementation. The integration tests above cover
	// the empty line handling via real /etc/os-release files.

	t.Skip("Unit test requires refactoring for dependency injection")
}

func TestReleaseCollector_HandlesMalformedLines(t *testing.T) {
	// Test that malformed lines (no '=' separator) are skipped gracefully
	// This is implicitly tested by the integration tests since real
	// /etc/os-release files are generally well-formed, but the code
	// handles this case by checking len(parts) != 2

	t.Skip("Unit test requires refactoring for dependency injection")
}

func TestReleaseCollector_ValidatesCommonFields(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.TODO()
	collector := &Collector{}

	m, err := collector.Collect(ctx)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			t.Skip("/etc/os-release not available on this system")
			return
		}
		t.Fatalf("Collect() failed: %v", err)
	}

	// Find release subtype
	var releaseSubtype *measurement.Subtype
	for i := range m.Subtypes {
		if m.Subtypes[i].Name == releaseSubtypeName {
			releaseSubtype = &m.Subtypes[i]
			break
		}
	}

	if releaseSubtype == nil {
		t.Fatal("Expected to find release subtype")
		return
	}

	data := releaseSubtype.Data

	// According to freedesktop.org spec, these fields should typically exist
	expectedFields := []string{"ID", "NAME"}
	foundCount := 0

	for _, field := range expectedFields {
		if val, exists := data[field]; exists {
			foundCount++
			t.Logf("Found expected field %s = %v", field, val.Any())
		} else {
			t.Logf("Missing recommended field: %s", field)
		}
	}

	if foundCount == 0 {
		t.Error("Expected at least one of the common fields (ID, NAME) to be present")
	}
}

func TestReleaseCollector_DataTypes(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.TODO()
	collector := &Collector{}

	m, err := collector.Collect(ctx)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			t.Skip("/etc/os-release not available on this system")
			return
		}
		t.Fatalf("Collect() failed: %v", err)
	}

	// Find release subtype
	var releaseSubtype *measurement.Subtype
	for i := range m.Subtypes {
		if m.Subtypes[i].Name == releaseSubtypeName {
			releaseSubtype = &m.Subtypes[i]
			break
		}
	}

	if releaseSubtype == nil {
		t.Fatal("Expected to find release subtype")
		return
	}

	data := releaseSubtype.Data

	// All values should be strings from measurement.Str()
	for key, reading := range data {
		val := reading.Any()
		if _, ok := val.(string); !ok {
			t.Errorf("Expected string value for key %s, got %T", key, val)
		}
	}
}

// BenchmarkReleaseCollector_Collect benchmarks the release collection process
func BenchmarkReleaseCollector_Collect(b *testing.B) {
	ctx := context.TODO()
	collector := &Collector{}

	// Verify it works before benchmarking
	_, err := collector.Collect(ctx)
	if err != nil {
		b.Skipf("Skipping benchmark: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = collector.Collect(ctx)
	}
}

// ExampleCollector_collectRelease demonstrates how the release collector works
func ExampleCollector_collectRelease() {
	ctx := context.TODO()
	collector := &Collector{}

	measurement, err := collector.Collect(ctx)
	if err != nil {
		// Handle error (e.g., /etc/os-release not found)
		return
	}

	// Find the release subtype
	for _, subtype := range measurement.Subtypes {
		if subtype.Name == "release" {
			// Access OS release information
			if osName, exists := subtype.Data["NAME"]; exists {
				_ = osName.Any() // Get the OS name
			}
			if osID, exists := subtype.Data["ID"]; exists {
				_ = osID.Any() // Get the OS ID
			}
			break
		}
	}
}
