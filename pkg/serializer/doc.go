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

// Package serializer provides encoding and decoding of measurement data in multiple formats.
//
// # Overview
//
// The serializer package handles conversion between measurement data structures and
// various output formats including JSON, YAML, and human-readable tables. It supports
// both encoding (writing data) and decoding (reading data) operations with automatic
// format detection.
//
// # Supported Formats
//
// JSON:
//   - Machine-parseable, compact representation
//   - Suitable for API responses and programmatic consumption
//   - Standard encoding/json package
//
// YAML:
//   - Human-readable with preserved structure
//   - Suitable for configuration files and version control
//   - gopkg.in/yaml.v3 package
//
// Table:
//   - Hierarchical text representation
//   - Suitable for terminal/console viewing
//   - Custom tree-style formatting
//   - Read-only (no deserialization support)
//
// # Core Types
//
// Format: Enum representing output formats (JSON, YAML, Table)
//
// Serializer: Interface for encoding data to output
//
//	type Serializer interface {
//	    Serialize(v any) error
//	}
//
// Reader: Handles decoding data from input sources
//
//	type Reader struct {
//	    format Format
//	    input  io.Reader
//	    closer io.Closer
//	}
//
// # Usage - Encoding
//
// Write to stdout (YAML):
//
//	serializer, err := serializer.NewStdoutSerializer(serializer.FormatYAML)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	data := map[string]any{"version": "1.0.0", "status": "ok"}
//	if err := serializer.Serialize(data); err != nil {
//	    log.Fatal(err)
//	}
//
// Write to file with automatic format detection:
//
//	serializer, err := serializer.NewFileSerializer("config.json")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer serializer.Close()
//
//	if err := serializer.Serialize(data); err != nil {
//	    log.Fatal(err)
//	}
//
// Write with explicit format:
//
//	serializer, err := serializer.NewFileSerializerWithFormat("output.txt", serializer.FormatTable)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer serializer.Close()
//
//	snapshot := // ... measurement data
//	if err := serializer.Serialize(snapshot); err != nil {
//	    log.Fatal(err)
//	}
//
// # Usage - Decoding
//
// Read from file with automatic format detection:
//
//	reader, err := serializer.NewFileReader("snapshot.yaml")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer reader.Close()
//
//	var snapshot snapshotter.Snapshot
//	if err := reader.Read(&snapshot); err != nil {
//	    log.Fatal(err)
//	}
//
// Read from stdin:
//
//	reader, err := serializer.NewStdinReader(serializer.FormatJSON)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	var data map[string]any
//	if err := reader.Read(&data); err != nil {
//	    log.Fatal(err)
//	}
//
// Read with custom io.Reader:
//
//	reader, err := serializer.NewReader(serializer.FormatYAML, strings.NewReader(yamlData))
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	var config Config
//	if err := reader.Read(&config); err != nil {
//	    log.Fatal(err)
//	}
//
// # Format Detection
//
// File extension-based detection:
//   - .json → JSON
//   - .yaml, .yml → YAML
//   - .table, .txt → Table
//   - Other → JSON (default)
//
// Format detection is automatic when using:
//   - NewFileSerializer(path)
//   - NewFileReader(path)
//
// # Table Format
//
// The table format provides hierarchical visualization:
//
//	Snapshot
//	├─ version: v1.0.0
//	├─ measurements:
//	│  ├─ K8s
//	│  │  ├─ server
//	│  │  │  ├─ version: 1.33.5
//	│  │  │  └─ platform: linux/amd64
//	│  │  └─ node
//	│  │     ├─ provider: eks
//	│  │     └─ kernel: 6.8.0
//	│  └─ GPU
//	│     ├─ driver: 570.158.01
//	│     └─ model: H100
//
// Table format:
//   - Does not support deserialization (read-only)
//   - Best for human viewing in terminals
//   - Preserves structure with tree-style indentation
//
// # Resource Management
//
// Always close serializers and readers that manage files:
//
//	serializer, err := serializer.NewFileSerializer("output.json")
//	if err != nil {
//	    return err
//	}
//	defer serializer.Close()  // Required for file resources
//
// Stdout/stdin serializers don't require closing but Close() is safe to call.
//
// # Error Handling
//
// Errors are returned when:
//   - Format is unknown or unsupported
//   - File cannot be opened or created
//   - Data cannot be marshaled/unmarshaled
//   - Table format used for deserialization
//
// All errors include context for debugging.
//
// # Integration
//
// Used throughout CNS for data I/O:
//   - pkg/cli - Command output formatting
//   - pkg/snapshotter - Snapshot serialization
//   - pkg/api - HTTP response encoding
//   - pkg/recipe - Recipe output
package serializer
