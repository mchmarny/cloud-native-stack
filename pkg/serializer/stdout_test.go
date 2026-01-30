// Copyright (c) 2025, NVIDIA CORPORATION.  All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package serializer

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os"
	"testing"
)

func TestStdoutSerializer_Serialize(t *testing.T) {
	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	defer func() {
		os.Stdout = oldStdout
	}()

	serializer := &StdoutSerializer{}
	data := map[string]any{
		"key":   "value",
		"count": 42,
	}

	err := serializer.Serialize(context.Background(), data)
	if err != nil {
		t.Fatalf("Serialize failed: %v", err)
	}

	// Close writer and read captured output
	w.Close()
	var buf bytes.Buffer
	io.Copy(&buf, r)

	// Verify it's valid JSON
	var result map[string]any
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("failed to unmarshal output: %v", err)
	}

	if result["key"] != "value" {
		t.Errorf("expected key=value, got %v", result["key"])
	}

	if result["count"].(float64) != 42 {
		t.Errorf("expected count=42, got %v", result["count"])
	}
}

func TestStdoutSerializer_MarshalError(t *testing.T) {
	serializer := &StdoutSerializer{}

	// Channel cannot be marshaled to JSON
	badData := make(chan int)

	err := serializer.Serialize(context.Background(), badData)
	if err == nil {
		t.Fatal("expected error for unmarshalable data")
	}

	if err.Error() == "" {
		t.Error("expected error message")
	}
}

func TestStdoutSerializer_NilData(t *testing.T) {
	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	defer func() {
		os.Stdout = oldStdout
	}()

	serializer := &StdoutSerializer{}

	err := serializer.Serialize(context.Background(), nil)
	if err != nil {
		t.Fatalf("Serialize failed for nil: %v", err)
	}

	// Close writer and read captured output
	w.Close()
	var buf bytes.Buffer
	io.Copy(&buf, r)

	// nil should serialize to "null"
	output := buf.String()
	if output != "null\n" && output != "null\r\n" {
		t.Errorf("expected 'null', got %q", output)
	}
}

func TestStdoutSerializer_ComplexStructure(t *testing.T) {
	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	defer func() {
		os.Stdout = oldStdout
	}()

	serializer := &StdoutSerializer{}

	type nested struct {
		Field string `json:"field"`
	}

	data := struct {
		Name   string `json:"name"`
		Items  []int  `json:"items"`
		Nested nested `json:"nested"`
		Active bool   `json:"active"`
	}{
		Name:   "test",
		Items:  []int{1, 2, 3},
		Nested: nested{Field: "value"},
		Active: true,
	}

	err := serializer.Serialize(context.Background(), data)
	if err != nil {
		t.Fatalf("Serialize failed: %v", err)
	}

	// Close writer and read captured output
	w.Close()
	var buf bytes.Buffer
	io.Copy(&buf, r)

	// Verify structure
	var result struct {
		Name   string `json:"name"`
		Items  []int  `json:"items"`
		Nested nested `json:"nested"`
		Active bool   `json:"active"`
	}

	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("failed to unmarshal output: %v", err)
	}

	if result.Name != "test" {
		t.Errorf("expected name=test, got %s", result.Name)
	}

	if len(result.Items) != 3 {
		t.Errorf("expected 3 items, got %d", len(result.Items))
	}

	if result.Nested.Field != "value" {
		t.Errorf("expected nested.field=value, got %s", result.Nested.Field)
	}

	if !result.Active {
		t.Error("expected active=true")
	}
}

func TestStdoutSerializer_EmptyStruct(t *testing.T) {
	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	defer func() {
		os.Stdout = oldStdout
	}()

	serializer := &StdoutSerializer{}
	data := struct{}{}

	err := serializer.Serialize(context.Background(), data)
	if err != nil {
		t.Fatalf("Serialize failed: %v", err)
	}

	// Close writer and read captured output
	w.Close()
	var buf bytes.Buffer
	io.Copy(&buf, r)

	// Empty struct serializes to {}
	var result map[string]any
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("failed to unmarshal output: %v", err)
	}

	if len(result) != 0 {
		t.Errorf("expected empty object, got %v", result)
	}
}
