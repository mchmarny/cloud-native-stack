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

package measurement

import (
	"testing"
)

func TestCompare(t *testing.T) {
	tests := []struct {
		name    string
		m1      Measurement
		m2      Measurement
		wantLen int
		wantErr bool
		check   func(t *testing.T, diffs []*Subtype)
	}{
		{
			name: "different types",
			m1: Measurement{
				Type: TypeK8s,
				Subtypes: []Subtype{
					{Name: "cluster", Data: map[string]Reading{"version": Str("1.28.0")}},
				},
			},
			m2: Measurement{
				Type: TypeGPU,
				Subtypes: []Subtype{
					{Name: "gpu", Data: map[string]Reading{"driver": Str("535.104.05")}},
				},
			},
			wantErr: true,
		},
		{
			name: "identical measurements",
			m1: Measurement{
				Type: TypeK8s,
				Subtypes: []Subtype{
					{Name: "cluster", Data: map[string]Reading{"version": Str("1.28.0"), "nodes": Int(3)}},
				},
			},
			m2: Measurement{
				Type: TypeK8s,
				Subtypes: []Subtype{
					{Name: "cluster", Data: map[string]Reading{"version": Str("1.28.0"), "nodes": Int(3)}},
				},
			},
			wantLen: 0,
		},
		{
			name: "new subtype in m2",
			m1: Measurement{
				Type: TypeK8s,
				Subtypes: []Subtype{
					{Name: "cluster", Data: map[string]Reading{"version": Str("1.28.0")}},
				},
			},
			m2: Measurement{
				Type: TypeK8s,
				Subtypes: []Subtype{
					{Name: "cluster", Data: map[string]Reading{"version": Str("1.28.0")}},
					{Name: "node", Data: map[string]Reading{"count": Int(5)}},
				},
			},
			wantLen: 1,
			check: func(t *testing.T, diffs []*Subtype) {
				if diffs[0].Name != "node" {
					t.Errorf("Expected subtype name 'node', got %q", diffs[0].Name)
				}
				if len(diffs[0].Data) != 1 {
					t.Errorf("Expected 1 data item, got %d", len(diffs[0].Data))
				}
			},
		},
		{
			name: "changed value in existing subtype",
			m1: Measurement{
				Type: TypeK8s,
				Subtypes: []Subtype{
					{Name: "cluster", Data: map[string]Reading{"version": Str("1.28.0"), "nodes": Int(3)}},
				},
			},
			m2: Measurement{
				Type: TypeK8s,
				Subtypes: []Subtype{
					{Name: "cluster", Data: map[string]Reading{"version": Str("1.29.0"), "nodes": Int(3)}},
				},
			},
			wantLen: 1,
			check: func(t *testing.T, diffs []*Subtype) {
				if diffs[0].Name != testSubtypeCluster {
					t.Errorf("Expected subtype name %q, got %q", testSubtypeCluster, diffs[0].Name)
				}
				if len(diffs[0].Data) != 1 {
					t.Errorf("Expected 1 data item (only changed value), got %d", len(diffs[0].Data))
				}
				v, err := diffs[0].GetString("version")
				if err != nil {
					t.Errorf("Error getting version: %v", err)
				}
				if v != "1.29.0" {
					t.Errorf("Expected version '1.29.0', got %q", v)
				}
			},
		},
		{
			name: "new key in existing subtype",
			m1: Measurement{
				Type: TypeK8s,
				Subtypes: []Subtype{
					{Name: "cluster", Data: map[string]Reading{"version": Str("1.28.0")}},
				},
			},
			m2: Measurement{
				Type: TypeK8s,
				Subtypes: []Subtype{
					{Name: "cluster", Data: map[string]Reading{"version": Str("1.28.0"), "nodes": Int(3)}},
				},
			},
			wantLen: 1,
			check: func(t *testing.T, diffs []*Subtype) {
				if len(diffs[0].Data) != 1 {
					t.Errorf("Expected 1 data item (new key), got %d", len(diffs[0].Data))
				}
				if !diffs[0].Has("nodes") {
					t.Error("Expected 'nodes' key in diff")
				}
			},
		},
		{
			name: "multiple changes",
			m1: Measurement{
				Type: TypeK8s,
				Subtypes: []Subtype{
					{Name: "cluster", Data: map[string]Reading{"version": Str("1.28.0"), "nodes": Int(3)}},
					{Name: "pod", Data: map[string]Reading{"ready": Bool(true)}},
				},
			},
			m2: Measurement{
				Type: TypeK8s,
				Subtypes: []Subtype{
					{Name: "cluster", Data: map[string]Reading{"version": Str("1.29.0"), "nodes": Int(5)}},
					{Name: "pod", Data: map[string]Reading{"ready": Bool(true)}},
					{Name: "service", Data: map[string]Reading{"count": Int(10)}},
				},
			},
			wantLen: 2,
			check: func(t *testing.T, diffs []*Subtype) {
				// Should have changes in 'cluster' and new 'service'
				names := make(map[string]bool)
				for _, d := range diffs {
					names[d.Name] = true
				}
				if !names["cluster"] {
					t.Error("Expected 'cluster' in diffs")
				}
				if !names["service"] {
					t.Error("Expected 'service' in diffs")
				}
			},
		},
		{
			name: "different numeric types with same value",
			m1: Measurement{
				Type: TypeK8s,
				Subtypes: []Subtype{
					{Name: "test", Data: map[string]Reading{"value": Int(42)}},
				},
			},
			m2: Measurement{
				Type: TypeK8s,
				Subtypes: []Subtype{
					{Name: "test", Data: map[string]Reading{"value": Int64(42)}},
				},
			},
			wantLen: 1,
			check: func(t *testing.T, diffs []*Subtype) {
				// Different types (int vs int64) should be detected as different
				if len(diffs[0].Data) != 1 {
					t.Errorf("Expected 1 data item, got %d", len(diffs[0].Data))
				}
			},
		},
		{
			name: "bool changes",
			m1: Measurement{
				Type: TypeK8s,
				Subtypes: []Subtype{
					{Name: "status", Data: map[string]Reading{"ready": Bool(false)}},
				},
			},
			m2: Measurement{
				Type: TypeK8s,
				Subtypes: []Subtype{
					{Name: "status", Data: map[string]Reading{"ready": Bool(true)}},
				},
			},
			wantLen: 1,
			check: func(t *testing.T, diffs []*Subtype) {
				v, err := diffs[0].GetBool("ready")
				if err != nil {
					t.Errorf("Error getting ready: %v", err)
				}
				if !v {
					t.Error("Expected ready to be true")
				}
			},
		},
		{
			name: "float changes",
			m1: Measurement{
				Type: TypeGPU,
				Subtypes: []Subtype{
					{Name: "gpu", Data: map[string]Reading{"temp": Float64(75.5)}},
				},
			},
			m2: Measurement{
				Type: TypeGPU,
				Subtypes: []Subtype{
					{Name: "gpu", Data: map[string]Reading{"temp": Float64(82.3)}},
				},
			},
			wantLen: 1,
			check: func(t *testing.T, diffs []*Subtype) {
				v, err := diffs[0].GetFloat64("temp")
				if err != nil {
					t.Errorf("Error getting temp: %v", err)
				}
				if v != 82.3 {
					t.Errorf("Expected temp 82.3, got %v", v)
				}
			},
		},
		{
			name: "empty m1 - all of m2 is new",
			m1: Measurement{
				Type:     TypeK8s,
				Subtypes: []Subtype{},
			},
			m2: Measurement{
				Type: TypeK8s,
				Subtypes: []Subtype{
					{Name: "cluster", Data: map[string]Reading{"version": Str("1.28.0")}},
					{Name: "node", Data: map[string]Reading{"count": Int(3)}},
				},
			},
			wantLen: 2,
		},
		{
			name: "empty m2 - no differences",
			m1: Measurement{
				Type: TypeK8s,
				Subtypes: []Subtype{
					{Name: "cluster", Data: map[string]Reading{"version": Str("1.28.0")}},
				},
			},
			m2: Measurement{
				Type:     TypeK8s,
				Subtypes: []Subtype{},
			},
			wantLen: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			diffs, err := Compare(tt.m1, tt.m2)
			if (err != nil) != tt.wantErr {
				t.Errorf("Compare() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}
			if len(diffs) != tt.wantLen {
				t.Errorf("Compare() returned %d diffs, want %d", len(diffs), tt.wantLen)
				return
			}
			if tt.check != nil {
				tt.check(t, diffs)
			}
		})
	}
}

func TestCompare_NoDiff_SameReferences(t *testing.T) {
	m := Measurement{
		Type: TypeK8s,
		Subtypes: []Subtype{
			{Name: "cluster", Data: map[string]Reading{"version": Str("1.28.0")}},
		},
	}

	diffs, err := Compare(m, m)
	if err != nil {
		t.Errorf("Compare() error = %v", err)
	}
	if len(diffs) != 0 {
		t.Errorf("Compare() returned %d diffs, want 0", len(diffs))
	}
}
