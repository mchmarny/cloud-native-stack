package serializers_test

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/NVIDIA/cloud-native-stack/pkg/serializers"
	"gopkg.in/yaml.v3"
)

type testConfig struct {
	Name  string
	Value int
}

func TestWriter_SerializeJSON(t *testing.T) {
	var buf bytes.Buffer
	writer := serializers.NewWriter(serializers.FormatJSON, &buf)

	data := []testConfig{
		{Name: "test1", Value: 123},
		{Name: "test2", Value: 456},
	}

	err := writer.Serialize(data)
	if err != nil {
		t.Fatalf("Serialize failed: %v", err)
	}

	// Verify it's valid JSON
	var result []testConfig
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	if len(result) != 2 {
		t.Errorf("Expected 2 items, got %d", len(result))
	}

	if result[0].Name != "test1" || result[0].Value != 123 {
		t.Errorf("Unexpected data: %+v", result[0])
	}
}

func TestWriter_SerializeYAML(t *testing.T) {
	var buf bytes.Buffer
	writer := serializers.NewWriter(serializers.FormatYAML, &buf)

	data := []testConfig{
		{Name: "test1", Value: 123},
		{Name: "test2", Value: 456},
	}

	err := writer.Serialize(data)
	if err != nil {
		t.Fatalf("Serialize failed: %v", err)
	}

	// Verify it's valid YAML
	var result []testConfig
	if err := yaml.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("Failed to unmarshal YAML: %v", err)
	}

	if len(result) != 2 {
		t.Errorf("Expected 2 items, got %d", len(result))
	}

	if result[0].Name != "test1" || result[0].Value != 123 {
		t.Errorf("Unexpected data: %+v", result[0])
	}
}

func TestWriter_SerializeTable(t *testing.T) {
	var buf bytes.Buffer
	writer := serializers.NewWriter(serializers.FormatTable, &buf)

	data := []interface{}{
		testConfig{Name: "test1", Value: 123},
		testConfig{Name: "test2", Value: 456},
	}

	err := writer.Serialize(data)
	if err != nil {
		t.Fatalf("Serialize failed: %v", err)
	}

	output := buf.String()

	// Verify output contains expected elements
	if !strings.Contains(output, "FIELD") || !strings.Contains(output, "VALUE") {
		t.Error("Expected table header not found")
	}

	if !strings.Contains(output, "[0].Name") || !strings.Contains(output, "[1].Value") {
		t.Error("Expected flattened keys not found")
	}
}

func TestWriter_UnsupportedFormat(t *testing.T) {
	var buf bytes.Buffer
	writer := serializers.NewWriter("invalid", &buf)

	err := writer.Serialize([]testConfig{})
	if err == nil {
		t.Error("Expected error for unsupported format")
	}

	if !strings.Contains(err.Error(), "unsupported format") {
		t.Errorf("Unexpected error message: %v", err)
	}
}

func TestWriter_NilOutput(t *testing.T) {
	// Should default to stdout
	writer := serializers.NewStdoutWriter(serializers.FormatJSON)

	if writer == nil {
		t.Fatal("Expected non-nil writer")
	}

	// Don't actually run Serialize as it would write to stdout
}

func TestNewWriter_DefaultsToStdout(t *testing.T) {
	writer := serializers.NewStdoutWriter(serializers.FormatJSON)
	if writer == nil {
		t.Fatal("Expected non-nil writer with nil output")
	}
}
