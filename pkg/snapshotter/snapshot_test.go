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

package snapshotter

import (
	"context"
	"fmt"
	"testing"

	"github.com/NVIDIA/cloud-native-stack/pkg/collector"
	"github.com/NVIDIA/cloud-native-stack/pkg/header"
	"github.com/NVIDIA/cloud-native-stack/pkg/measurement"
)

func TestNewSnapshot(t *testing.T) {
	snap := NewSnapshot()

	if snap == nil {
		t.Fatal("NewSnapshot() returned nil")
		return
	}

	if snap.Measurements == nil {
		t.Error("Measurements should be initialized")
	}

	if len(snap.Measurements) != 0 {
		t.Errorf("Measurements length = %d, want 0", len(snap.Measurements))
	}
}

func TestNodeSnapshotter_Measure(t *testing.T) {
	t.Run("with nil factory uses default", func(t *testing.T) {
		snapshotter := &NodeSnapshotter{
			Version:    "1.0.0",
			Factory:    nil, // Will use default
			Serializer: &mockSerializer{},
		}

		ctx := context.Background()
		err := snapshotter.Measure(ctx)

		// This will fail because default factory requires actual system resources
		// But we verify that Factory is set
		if snapshotter.Factory == nil {
			t.Error("Factory should be set to default when nil")
		}

		// Error is expected since we don't have real collectors
		if err == nil {
			t.Log("Measure succeeded (unexpected in test environment)")
		}
	})

	t.Run("with mock factory", func(t *testing.T) {
		factory := &mockFactory{}
		snapshotter := &NodeSnapshotter{
			Version:    "1.0.0",
			Factory:    factory,
			Serializer: &mockSerializer{},
		}

		ctx := context.Background()
		err := snapshotter.Measure(ctx)

		if err != nil {
			t.Errorf("Measure() error = %v, want nil", err)
		}

		if !factory.k8sCalled {
			t.Error("Kubernetes collector not called")
		}

		if !factory.systemdCalled {
			t.Error("SystemD collector not called")
		}

		if !factory.osCalled {
			t.Error("OS collector not called")
		}
	})

	t.Run("handles collector errors", func(t *testing.T) {
		factory := &mockFactory{
			k8sError: fmt.Errorf("k8s error"),
		}
		snapshotter := &NodeSnapshotter{
			Version:    "1.0.0",
			Factory:    factory,
			Serializer: &mockSerializer{},
		}

		ctx := context.Background()
		err := snapshotter.Measure(ctx)

		if err == nil {
			t.Error("Measure() should return error when collector fails")
		}
	})
}

func TestSnapshot_Init(t *testing.T) {
	snap := NewSnapshot()
	snap.Init(header.KindSnapshot, FullAPIVersion, "1.0.0")

	if snap.Kind != header.KindSnapshot {
		t.Errorf("Kind = %s, want %s", snap.Kind, header.KindSnapshot)
	}

	if snap.Metadata == nil {
		t.Error("Metadata should be initialized")
	}
}

func TestParseNodeSelectors(t *testing.T) {
	tests := []struct {
		name      string
		selectors []string
		want      map[string]string
		wantErr   bool
	}{
		{
			name:      "empty selectors",
			selectors: []string{},
			want:      map[string]string{},
			wantErr:   false,
		},
		{
			name:      "single selector",
			selectors: []string{"kubernetes.io/os=linux"},
			want:      map[string]string{"kubernetes.io/os": "linux"},
			wantErr:   false,
		},
		{
			name:      "multiple selectors",
			selectors: []string{"env=prod", "tier=backend"},
			want:      map[string]string{"env": "prod", "tier": "backend"},
			wantErr:   false,
		},
		{
			name:      "invalid format no equals",
			selectors: []string{"invalid"},
			wantErr:   true,
		},
		{
			name:      "value with equals sign",
			selectors: []string{"key=value=with=equals"},
			want:      map[string]string{"key": "value=with=equals"},
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseNodeSelectors(tt.selectors)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseNodeSelectors() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if len(got) != len(tt.want) {
					t.Errorf("ParseNodeSelectors() len = %v, want %v", len(got), len(tt.want))
				}
				for k, v := range tt.want {
					if got[k] != v {
						t.Errorf("ParseNodeSelectors()[%v] = %v, want %v", k, got[k], v)
					}
				}
			}
		})
	}
}

func TestParseTolerations(t *testing.T) {
	tests := []struct {
		name        string
		tolerations []string
		wantLen     int
		wantErr     bool
	}{
		{
			name:        "empty tolerations returns defaults",
			tolerations: []string{},
			wantLen:     1, // Default "tolerate all" toleration
			wantErr:     false,
		},
		{
			name:        "key=value:effect",
			tolerations: []string{"key=value:NoSchedule"},
			wantLen:     1,
			wantErr:     false,
		},
		{
			name:        "key:effect (exists)",
			tolerations: []string{"key:NoExecute"},
			wantLen:     1,
			wantErr:     false,
		},
		{
			name:        "multiple tolerations",
			tolerations: []string{"key1=val1:NoSchedule", "key2:NoExecute"},
			wantLen:     2,
			wantErr:     false,
		},
		{
			name:        "invalid format no colon",
			tolerations: []string{"invalid"},
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseTolerations(tt.tolerations)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseTolerations() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && len(got) != tt.wantLen {
				t.Errorf("ParseTolerations() len = %v, want %v", len(got), tt.wantLen)
			}
		})
	}
}

// Mock implementations for testing

type mockSerializer struct {
	serialized bool
	data       any
}

func (m *mockSerializer) Serialize(ctx context.Context, data any) error {
	m.serialized = true
	m.data = data
	return nil
}

type mockFactory struct {
	k8sCalled     bool
	systemdCalled bool
	osCalled      bool
	gpuCalled     bool

	k8sError     error
	systemdError error
	osError      error
	gpuError     error
}

func (m *mockFactory) CreateKubernetesCollector() collector.Collector {
	m.k8sCalled = true
	return &mockCollector{err: m.k8sError}
}

func (m *mockFactory) CreateSystemDCollector() collector.Collector {
	m.systemdCalled = true
	return &mockCollector{err: m.systemdError}
}

func (m *mockFactory) CreateOSCollector() collector.Collector {
	m.osCalled = true
	return &mockCollector{err: m.osError}
}

func (m *mockFactory) CreateGPUCollector() collector.Collector {
	m.gpuCalled = true
	return &mockCollector{err: m.gpuError}
}

type mockCollector struct {
	err error
}

func (m *mockCollector) Collect(ctx context.Context) (*measurement.Measurement, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &measurement.Measurement{
		Type:     measurement.TypeK8s,
		Subtypes: []measurement.Subtype{},
	}, nil
}
