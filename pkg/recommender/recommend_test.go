package recommender

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/NVIDIA/cloud-native-stack/pkg/header"
	"github.com/NVIDIA/cloud-native-stack/pkg/measurement"
	"github.com/NVIDIA/cloud-native-stack/pkg/recipe"
	"github.com/NVIDIA/cloud-native-stack/pkg/snapshotter"
	"github.com/NVIDIA/cloud-native-stack/pkg/version"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name    string
		opts    []Option
		wantVer string
	}{
		{
			name:    "default",
			opts:    nil,
			wantVer: "",
		},
		{
			name:    "with version",
			opts:    []Option{WithVersion("v1.2.3")},
			wantVer: "v1.2.3",
		},
		{
			name:    "multiple options",
			opts:    []Option{WithVersion("v1.0.0"), WithVersion("v2.0.0")},
			wantVer: "v2.0.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := New(tt.opts...)
			if got.Version != tt.wantVer {
				t.Errorf("New() version = %v, want %v", got.Version, tt.wantVer)
			}
		})
	}
}

func TestConfigRecommender_Recommend(t *testing.T) {
	// Create a valid test snapshot
	validSnapshot := createTestSnapshot()

	tests := []struct {
		name      string
		intent    recipe.IntentType
		snapshot  *snapshotter.Snapshot
		wantError bool
		errMsg    string
	}{
		{
			name:      "valid snapshot and intent",
			intent:    recipe.IntentTraining,
			snapshot:  validSnapshot,
			wantError: false,
		},
		{
			name:      "nil snapshot",
			intent:    recipe.IntentTraining,
			snapshot:  nil,
			wantError: true,
			errMsg:    "snapshot cannot be nil",
		},
		{
			name:      "snapshot with no measurements",
			intent:    recipe.IntentTraining,
			snapshot:  snapshotter.NewSnapshot(),
			wantError: true,
			errMsg:    "snapshot contains no measurements",
		},
		{
			name:      "invalid intent",
			intent:    recipe.IntentType("invalid"),
			snapshot:  validSnapshot,
			wantError: true,
			errMsg:    "invalid intent type",
		},
		{
			name:      "inference intent",
			intent:    recipe.IntentInference,
			snapshot:  validSnapshot,
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := New(WithVersion("v1.0.0"))
			ctx := context.Background()

			got, err := r.Recommend(ctx, tt.intent, tt.snapshot)

			if tt.wantError {
				if err == nil {
					t.Errorf("Recommend() expected error but got nil")
					return
				}
				if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("Recommend() error = %v, want error containing %v", err, tt.errMsg)
				}
				return
			}

			if err != nil {
				t.Errorf("Recommend() unexpected error = %v", err)
				return
			}

			if got == nil {
				t.Error("Recommend() returned nil recipe")
				return
			}

			// Verify metadata
			if got.Metadata == nil {
				t.Error("Recommend() recipe has nil metadata")
				return
			}

			if got.Metadata["recommender-version"] != "v1.0.0" {
				t.Errorf("Recommend() recommender-version = %v, want v1.0.0",
					got.Metadata["recommender-version"])
			}
		})
	}
}

func TestConfigRecommender_Recommend_ContextCancellation(t *testing.T) {
	r := New(WithVersion("v1.0.0"))
	snapshot := createTestSnapshot()

	// Create a canceled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := r.Recommend(ctx, recipe.IntentTraining, snapshot)
	if err == nil {
		t.Error("Recommend() expected error with canceled context but got nil")
	}
}

func TestConfigRecommender_Recommend_Timeout(t *testing.T) {
	r := New(WithVersion("v1.0.0"))
	snapshot := createTestSnapshot()

	// Create a context with a very short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()

	// Give the context time to expire
	time.Sleep(10 * time.Millisecond)

	_, err := r.Recommend(ctx, recipe.IntentTraining, snapshot)
	if err == nil {
		t.Error("Recommend() expected timeout error but got nil")
	}
}

func TestQueryFromSnapshot(t *testing.T) {
	tests := []struct {
		name      string
		intent    recipe.IntentType
		snapshot  *snapshotter.Snapshot
		wantError bool
		errMsg    string
		validate  func(*testing.T, *recipe.Query)
	}{
		{
			name:      "nil snapshot",
			intent:    recipe.IntentTraining,
			snapshot:  nil,
			wantError: true,
			errMsg:    "snapshot cannot be nil",
		},
		{
			name:      "empty measurements",
			intent:    recipe.IntentTraining,
			snapshot:  snapshotter.NewSnapshot(),
			wantError: true,
			errMsg:    "snapshot contains no measurements",
		},
		{
			name:     "complete snapshot",
			intent:   recipe.IntentTraining,
			snapshot: createTestSnapshot(),
			validate: func(t *testing.T, q *recipe.Query) {
				if q.Intent != recipe.IntentTraining {
					t.Errorf("Intent = %v, want %v", q.Intent, recipe.IntentTraining)
				}
				if !q.IncludeContext {
					t.Error("IncludeContext should be true by default")
				}
				if q.Os != recipe.OSUbuntu {
					t.Errorf("Os = %v, want %v", q.Os, recipe.OSUbuntu)
				}
				if q.GPU != recipe.GPUH100 {
					t.Errorf("GPU = %v, want %v", q.GPU, recipe.GPUH100)
				}
				if q.Service != recipe.ServiceEKS {
					t.Errorf("Service = %v, want %v", q.Service, recipe.ServiceEKS)
				}
			},
		},
		{
			name:     "snapshot with partial data",
			intent:   recipe.IntentInference,
			snapshot: createPartialSnapshot(),
			validate: func(t *testing.T, q *recipe.Query) {
				if q.Intent != recipe.IntentInference {
					t.Errorf("Intent = %v, want %v", q.Intent, recipe.IntentInference)
				}
				if q.Os != recipe.OSUbuntu {
					t.Errorf("Os = %v, want %v", q.Os, recipe.OSUbuntu)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := QueryFromSnapshot(tt.intent, tt.snapshot)

			if tt.wantError {
				if err == nil {
					t.Error("QueryFromSnapshot() expected error but got nil")
					return
				}
				if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("QueryFromSnapshot() error = %v, want error containing %v", err, tt.errMsg)
				}
				return
			}

			if err != nil {
				t.Errorf("QueryFromSnapshot() unexpected error = %v", err)
				return
			}

			if got == nil {
				t.Error("QueryFromSnapshot() returned nil query")
				return
			}

			if tt.validate != nil {
				tt.validate(t, got)
			}
		})
	}
}

func TestParseOSSubtypes(t *testing.T) {
	tests := []struct {
		name      string
		subtypes  []measurement.Subtype
		wantError bool
		validate  func(*testing.T, *recipe.Query)
	}{
		{
			name: "complete OS data",
			subtypes: []measurement.Subtype{
				{
					Name: "sysctl",
					Data: map[string]measurement.Reading{
						"/proc/sys/kernel/osrelease": measurement.Str("5.15.0-1028-aws"),
					},
				},
				{
					Name: "release",
					Data: map[string]measurement.Reading{
						"ID":         measurement.Str("ubuntu"),
						"VERSION_ID": measurement.Str("22.04"),
					},
				},
			},
			validate: func(t *testing.T, q *recipe.Query) {
				if q.Os != recipe.OSUbuntu {
					t.Errorf("Os = %v, want %v", q.Os, recipe.OSUbuntu)
				}
				if q.OsVersion == nil || q.OsVersion.Major != 22 || q.OsVersion.Minor != 4 {
					t.Errorf("OsVersion = %v, want 22.4", q.OsVersion)
				}
				if q.Kernel == nil || q.Kernel.Major != 5 || q.Kernel.Minor != 15 || q.Kernel.Patch != 0 {
					t.Errorf("Kernel = %v, want 5.15.0", q.Kernel)
				}
			},
		},
		{
			name: "invalid kernel version",
			subtypes: []measurement.Subtype{
				{
					Name: "sysctl",
					Data: map[string]measurement.Reading{
						"/proc/sys/kernel/osrelease": measurement.Str("invalid"),
					},
				},
			},
			wantError: true,
		},
		{
			name: "invalid OS family",
			subtypes: []measurement.Subtype{
				{
					Name: "release",
					Data: map[string]measurement.Reading{
						"ID": measurement.Str("invalid-os"),
					},
				},
			},
			wantError: true,
		},
		{
			name: "missing OS data",
			subtypes: []measurement.Subtype{
				{
					Name: "other",
					Data: map[string]measurement.Reading{},
				},
			},
			validate: func(t *testing.T, q *recipe.Query) {
				// Should not populate fields if data is missing
				if q.Os != "" && q.Os != recipe.OSAny {
					t.Errorf("Os should be empty, got %v", q.Os)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q := &recipe.Query{}
			err := parseOSSubtypes(tt.subtypes, q)

			if tt.wantError {
				if err == nil {
					t.Error("parseOSSubtypes() expected error but got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("parseOSSubtypes() unexpected error = %v", err)
				return
			}

			if tt.validate != nil {
				tt.validate(t, q)
			}
		})
	}
}

func TestParseK8sSubtypes(t *testing.T) {
	tests := []struct {
		name      string
		subtypes  []measurement.Subtype
		wantError bool
		validate  func(*testing.T, *recipe.Query)
	}{
		{
			name: "complete K8s data",
			subtypes: []measurement.Subtype{
				{
					Name: "server",
					Data: map[string]measurement.Reading{
						"version": measurement.Str("v1.28.0-eks-3025e55"),
					},
				},
				{
					Name: "node",
					Data: map[string]measurement.Reading{
						"provider": measurement.Str("eks"),
					},
				},
			},
			validate: func(t *testing.T, q *recipe.Query) {
				if q.K8s == nil || q.K8s.String() != "1.28.0" {
					t.Errorf("K8s = %v, want 1.28.0", q.K8s)
				}
				if q.Service != recipe.ServiceEKS {
					t.Errorf("Service = %v, want %v", q.Service, recipe.ServiceEKS)
				}
			},
		},
		{
			name: "GKE service",
			subtypes: []measurement.Subtype{
				{
					Name: "node",
					Data: map[string]measurement.Reading{
						"provider": measurement.Str("gke"),
					},
				},
			},
			validate: func(t *testing.T, q *recipe.Query) {
				if q.Service != recipe.ServiceGKE {
					t.Errorf("Service = %v, want %v", q.Service, recipe.ServiceGKE)
				}
			},
		},
		{
			name: "invalid K8s version",
			subtypes: []measurement.Subtype{
				{
					Name: "server",
					Data: map[string]measurement.Reading{
						"version": measurement.Str("invalid"),
					},
				},
			},
			wantError: true,
		},
		{
			name: "invalid service type",
			subtypes: []measurement.Subtype{
				{
					Name: "node",
					Data: map[string]measurement.Reading{
						"provider": measurement.Str("invalid-service"),
					},
				},
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q := &recipe.Query{}
			err := parseK8sSubtypes(tt.subtypes, q)

			if tt.wantError {
				if err == nil {
					t.Error("parseK8sSubtypes() expected error but got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("parseK8sSubtypes() unexpected error = %v", err)
				return
			}

			if tt.validate != nil {
				tt.validate(t, q)
			}
		})
	}
}

func TestParseGPUSubtypes(t *testing.T) {
	tests := []struct {
		name      string
		subtypes  []measurement.Subtype
		wantError bool
		validate  func(*testing.T, *recipe.Query)
	}{
		{
			name: "H100 GPU",
			subtypes: []measurement.Subtype{
				{
					Name: "smi",
					Data: map[string]measurement.Reading{
						"gpu.model": measurement.Str("NVIDIA H100 PCIe"),
					},
				},
			},
			validate: func(t *testing.T, q *recipe.Query) {
				if q.GPU != recipe.GPUH100 {
					t.Errorf("GPU = %v, want %v", q.GPU, recipe.GPUH100)
				}
			},
		},
		{
			name: "GB200 GPU",
			subtypes: []measurement.Subtype{
				{
					Name: "smi",
					Data: map[string]measurement.Reading{
						"gpu.model": measurement.Str("NVIDIA GB200"),
					},
				},
			},
			validate: func(t *testing.T, q *recipe.Query) {
				if q.GPU != recipe.GPUB200 {
					t.Errorf("GPU = %v, want %v", q.GPU, recipe.GPUB200)
				}
			},
		},
		{
			name: "unknown GPU model",
			subtypes: []measurement.Subtype{
				{
					Name: "smi",
					Data: map[string]measurement.Reading{
						"gpu.model": measurement.Str("NVIDIA A100"),
					},
				},
			},
			wantError: true, // Unknown models should be invalid
		},
		{
			name: "missing GPU data",
			subtypes: []measurement.Subtype{
				{
					Name: "other",
					Data: map[string]measurement.Reading{},
				},
			},
			validate: func(t *testing.T, q *recipe.Query) {
				if q.GPU != "" && q.GPU != recipe.GPUAny {
					t.Errorf("GPU should be empty, got %v", q.GPU)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q := &recipe.Query{}
			err := parseGPUSubtypes(tt.subtypes, q)

			if tt.wantError {
				if err == nil {
					t.Error("parseGPUSubtypes() expected error but got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("parseGPUSubtypes() unexpected error = %v", err)
				return
			}

			if tt.validate != nil {
				tt.validate(t, q)
			}
		})
	}
}

func TestWalkSubtypesForReading(t *testing.T) {
	subtypes := []measurement.Subtype{
		{
			Name: "sysctl",
			Data: map[string]measurement.Reading{
				"kernel.version": measurement.Str("5.15.0"),
				"vm.swappiness":  measurement.Int(10),
			},
		},
		{
			Name: "release",
			Data: map[string]measurement.Reading{
				"ID":         measurement.Str("ubuntu"),
				"VERSION_ID": measurement.Str("22.04"),
			},
		},
	}

	tests := []struct {
		name    string
		subtype string
		key     string
		wantNil bool
		wantVal string
	}{
		{
			name:    "existing key in first subtype",
			subtype: "sysctl",
			key:     "kernel.version",
			wantVal: "5.15.0",
		},
		{
			name:    "existing key in second subtype",
			subtype: "release",
			key:     "ID",
			wantVal: "ubuntu",
		},
		{
			name:    "non-existent key",
			subtype: "sysctl",
			key:     "nonexistent",
			wantNil: true,
		},
		{
			name:    "non-existent subtype",
			subtype: "nonexistent",
			key:     "kernel.version",
			wantNil: true,
		},
		{
			name:    "integer value",
			subtype: "sysctl",
			key:     "vm.swappiness",
			wantVal: "10",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := walkSubtypesForReading(tt.subtype, tt.key, subtypes)

			if tt.wantNil {
				if got != nil {
					t.Errorf("walkSubtypesForReading() = %v, want nil", got)
				}
				return
			}

			if got == nil {
				t.Error("walkSubtypesForReading() returned nil, want value")
				return
			}

			if got.String() != tt.wantVal {
				t.Errorf("walkSubtypesForReading() = %v, want %v", got.String(), tt.wantVal)
			}
		})
	}
}

func TestSubtypeQueryParsers(t *testing.T) {
	// Verify all measurement types have parsers
	expectedTypes := []measurement.Type{
		measurement.TypeOS,
		measurement.TypeK8s,
		measurement.TypeGPU,
	}

	for _, mt := range expectedTypes {
		t.Run(string(mt), func(t *testing.T) {
			parser, exists := subtypeQueryParsers[mt]
			if !exists {
				t.Errorf("No parser registered for measurement type %s", mt)
				return
			}
			if parser == nil {
				t.Errorf("Parser for measurement type %s is nil", mt)
			}
		})
	}
}

// Helper functions

func createTestSnapshot() *snapshotter.Snapshot {
	osVersion := version.MustParseVersion("22.04")
	kernelVersion := version.MustParseVersion("5.15.0-1028-aws")
	k8sVersion := version.MustParseVersion("v1.28.0-eks-3025e55")

	return &snapshotter.Snapshot{
		Header: header.Header{
			APIVersion: "v1",
			Kind:       "Snapshot",
			Metadata:   map[string]string{},
		},
		Measurements: []*measurement.Measurement{
			{
				Type: measurement.TypeOS,
				Subtypes: []measurement.Subtype{
					{
						Name: "sysctl",
						Data: map[string]measurement.Reading{
							"/proc/sys/kernel/osrelease": measurement.Str(kernelVersion.String()),
						},
					},
					{
						Name: "release",
						Data: map[string]measurement.Reading{
							"ID":         measurement.Str("ubuntu"),
							"VERSION_ID": measurement.Str(osVersion.String()),
						},
					},
				},
			},
			{
				Type: measurement.TypeK8s,
				Subtypes: []measurement.Subtype{
					{
						Name: "server",
						Data: map[string]measurement.Reading{
							"version": measurement.Str(k8sVersion.String()),
						},
					},
					{
						Name: "node",
						Data: map[string]measurement.Reading{
							"provider": measurement.Str("eks"),
						},
					},
				},
			},
			{
				Type: measurement.TypeGPU,
				Subtypes: []measurement.Subtype{
					{
						Name: "smi",
						Data: map[string]measurement.Reading{
							"gpu.model": measurement.Str("NVIDIA H100 PCIe"),
						},
					},
				},
			},
		},
	}
}

func createPartialSnapshot() *snapshotter.Snapshot {
	return &snapshotter.Snapshot{
		Header: header.Header{
			APIVersion: "v1",
			Kind:       "Snapshot",
			Metadata:   map[string]string{},
		},
		Measurements: []*measurement.Measurement{
			{
				Type: measurement.TypeOS,
				Subtypes: []measurement.Subtype{
					{
						Name: "release",
						Data: map[string]measurement.Reading{
							"ID": measurement.Str("ubuntu"),
						},
					},
				},
			},
		},
	}
}
