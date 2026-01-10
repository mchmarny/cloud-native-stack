package serializer

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

const (
	testName  = "test"
	test1Name = "test1"
)

func TestWriter_SerializeJSON(t *testing.T) {
	var buf bytes.Buffer
	writer := NewWriter(FormatJSON, &buf)

	data := []testConfig{
		{Name: test1Name, Value: 123},
		{Name: "test2", Value: 456},
	}

	err := writer.Serialize(context.Background(), data)
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

	if result[0].Name != test1Name || result[0].Value != 123 {
		t.Errorf("Unexpected data: %+v", result[0])
	}
}

func TestWriter_SerializeYAML(t *testing.T) {
	var buf bytes.Buffer
	writer := NewWriter(FormatYAML, &buf)

	data := []testConfig{
		{Name: test1Name, Value: 123},
		{Name: "test2", Value: 456},
	}

	err := writer.Serialize(context.Background(), data)
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

	if result[0].Name != test1Name || result[0].Value != 123 {
		t.Errorf("Unexpected data: %+v", result[0])
	}
}

func TestWriter_SerializeTable(t *testing.T) {
	var buf bytes.Buffer
	writer := NewWriter(FormatTable, &buf)

	data := []interface{}{
		testConfig{Name: test1Name, Value: 123},
		testConfig{Name: "test2", Value: 456},
	}

	err := writer.Serialize(context.Background(), data)
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
	// Note: NewWriter now defaults unknown formats to JSON instead of erroring
	// This test is kept to verify the fallback behavior
	var buf bytes.Buffer
	writer := NewWriter("invalid", &buf)

	if writer == nil {
		t.Fatal("Expected non-nil writer with unknown format")
	}

	// Should succeed because it falls back to JSON
	data := testConfig{Name: "test", Value: 123}
	err := writer.Serialize(context.Background(), data)
	if err != nil {
		t.Fatalf("Serialize should not fail with unknown format (falls back to JSON): %v", err)
	}

	// Verify it was serialized as JSON
	var result testConfig
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("Failed to unmarshal as JSON: %v", err)
	}

	if result.Name != testName || result.Value != 123 {
		t.Errorf("Unexpected data: %+v", result)
	}
}

func TestWriter_NilOutput(t *testing.T) {
	// Should default to stdout
	writer := NewStdoutWriter(FormatJSON)

	if writer == nil {
		t.Fatal("Expected non-nil writer")
	}

	// Don't actually run Serialize as it would write to stdout
}

func TestNewWriter_DefaultsToStdout(t *testing.T) {
	writer := NewStdoutWriter(FormatJSON)
	if writer == nil {
		t.Fatal("Expected non-nil writer with nil output")
	}
}

func TestWriter_Close(t *testing.T) {
	// Test closing stdout writer (should be safe)
	writer := NewStdoutWriter(FormatJSON)
	err := writer.Close()
	if err != nil {
		t.Errorf("Close on stdout writer should not error: %v", err)
	}

	// Test closing multiple times (should be safe)
	err = writer.Close()
	if err != nil {
		t.Errorf("Multiple Close calls should not error: %v", err)
	}
}

func TestNewFileWriterOrStdout_EmptyPath(t *testing.T) {
	tests := []string{"", "  ", "\t", "\n"}

	for _, path := range tests {
		writer := NewFileWriterOrStdout(FormatJSON, path)
		if writer == nil {
			t.Fatalf("Expected non-nil writer for empty path %q", path)
		}
		// Should default to stdout, so Close should be safe
		if closer, ok := writer.(Closer); ok {
			if err := closer.Close(); err != nil {
				t.Errorf("Close failed for empty path writer: %v", err)
			}
		}
	}
}

func TestNewFileWriterOrStdout_Success(t *testing.T) {
	// Create a temporary file path
	tmpFile := t.TempDir() + "/test_output.json"

	writer := NewFileWriterOrStdout(FormatJSON, tmpFile)
	if writer == nil {
		t.Fatal("Expected non-nil writer")
	}

	// Write some data
	data := testConfig{Name: testName, Value: 123}
	err := writer.Serialize(context.Background(), data)
	if err != nil {
		t.Fatalf("Serialize failed: %v", err)
	}

	// Close the writer
	if closer, ok := writer.(Closer); ok {
		err = closer.Close()
		if err != nil {
			t.Fatalf("Close failed: %v", err)
		}
	}

	// Verify file exists and has content
	content, err := os.ReadFile(tmpFile)
	if err != nil {
		t.Fatalf("Failed to read output file: %v", err)
	}

	if len(content) == 0 {
		t.Error("Expected file to have content")
	}

	// Verify it's valid JSON
	var result testConfig
	if err := json.Unmarshal(content, &result); err != nil {
		t.Fatalf("Failed to unmarshal file content: %v", err)
	}

	if result.Name != testName || result.Value != 123 {
		t.Errorf("Unexpected data in file: %+v", result)
	}
}

func TestNewFileWriterOrStdout_InvalidPath(t *testing.T) {
	// Try to create a file in a non-existent directory without creating it first
	writer := NewFileWriterOrStdout(FormatJSON, "/nonexistent/path/file.json")

	// Should fall back to stdout
	if writer == nil {
		t.Fatal("Expected non-nil writer (should fallback to stdout)")
	}

	// Close should be safe
	if closer, ok := writer.(Closer); ok {
		if err := closer.Close(); err != nil {
			t.Errorf("Close should not error on fallback writer: %v", err)
		}
	}
}

func TestFormat_IsUnknown(t *testing.T) {
	tests := []struct {
		format Format
		want   bool
	}{
		{FormatJSON, false},
		{FormatYAML, false},
		{FormatTable, false},
		{Format("invalid"), true},
		{Format("xml"), true},
		{Format(""), true},
	}

	for _, tt := range tests {
		t.Run(string(tt.format), func(t *testing.T) {
			if got := tt.format.IsUnknown(); got != tt.want {
				t.Errorf("Format(%q).IsUnknown() = %v, want %v", tt.format, got, tt.want)
			}
		})
	}
}

func TestNewWriter_UnknownFormat(t *testing.T) {
	var buf bytes.Buffer
	writer := NewWriter(Format("invalid"), &buf)

	if writer == nil {
		t.Fatal("Expected non-nil writer")
	}

	// Should default to JSON format
	data := testConfig{Name: "test", Value: 123}
	err := writer.Serialize(context.Background(), data)
	if err != nil {
		t.Fatalf("Serialize failed: %v", err)
	}

	// Verify it serialized as JSON
	var result testConfig
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("Failed to unmarshal as JSON: %v", err)
	}
}

func TestWriter_SerializeTable_EmptyData(t *testing.T) {
	var buf bytes.Buffer
	writer := NewWriter(FormatTable, &buf)

	// Empty slice
	err := writer.Serialize(context.Background(), []testConfig{})
	if err != nil {
		t.Fatalf("Serialize empty slice failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "<empty>") {
		t.Errorf("Expected '<empty>' in output for empty data, got: %s", output)
	}
}

func TestWriter_SerializeTable_NestedStructs(t *testing.T) {
	var buf bytes.Buffer
	writer := NewWriter(FormatTable, &buf)

	type inner struct {
		Field1 string
		Field2 int
	}

	type outer struct {
		Name  string
		Inner inner
	}

	data := outer{
		Name: "test",
		Inner: inner{
			Field1: "value",
			Field2: 42,
		},
	}

	err := writer.Serialize(context.Background(), data)
	if err != nil {
		t.Fatalf("Serialize failed: %v", err)
	}

	output := buf.String()

	// Should have flattened nested keys
	if !strings.Contains(output, "Inner.Field1") {
		t.Error("Expected flattened key 'Inner.Field1' not found")
	}

	if !strings.Contains(output, "Inner.Field2") {
		t.Error("Expected flattened key 'Inner.Field2' not found")
	}

	if !strings.Contains(output, "value") {
		t.Error("Expected value 'value' not found")
	}

	if !strings.Contains(output, "42") {
		t.Error("Expected value '42' not found")
	}
}

func TestWriter_SerializeTable_Maps(t *testing.T) {
	var buf bytes.Buffer
	writer := NewWriter(FormatTable, &buf)

	data := map[string]interface{}{
		"key1": "value1",
		"key2": 123,
		"key3": true,
	}

	err := writer.Serialize(context.Background(), data)
	if err != nil {
		t.Fatalf("Serialize failed: %v", err)
	}

	output := buf.String()

	// Should have all keys
	if !strings.Contains(output, "key1") || !strings.Contains(output, "key2") || !strings.Contains(output, "key3") {
		t.Error("Expected all keys in output")
	}
}

func TestWriter_SerializeTable_NilValues(t *testing.T) {
	var buf bytes.Buffer
	writer := NewWriter(FormatTable, &buf)

	type dataWithNil struct {
		Name  string
		Value *int
	}

	data := dataWithNil{
		Name:  "test",
		Value: nil,
	}

	err := writer.Serialize(context.Background(), data)
	if err != nil {
		t.Fatalf("Serialize failed: %v", err)
	}

	output := buf.String()

	// Should handle nil gracefully
	if !strings.Contains(output, "Name") {
		t.Error("Expected 'Name' field in output")
	}
}
