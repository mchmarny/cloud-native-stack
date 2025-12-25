package recipe

import (
	"testing"

	"github.com/NVIDIA/cloud-native-stack/pkg/version"
)

func TestQueryBuilder_Basic(t *testing.T) {
	query := NewQueryBuilder().
		WithOS(OSUbuntu).
		WithService(ServiceEKS).
		WithGPU(GPUH100).
		WithIntent(IntentTraining).
		Build()

	if query.Os != OSUbuntu {
		t.Errorf("expected Os %s, got %s", OSUbuntu, query.Os)
	}
	if query.Service != ServiceEKS {
		t.Errorf("expected Service %s, got %s", ServiceEKS, query.Service)
	}
	if query.GPU != GPUH100 {
		t.Errorf("expected GPU %s, got %s", GPUH100, query.GPU)
	}
	if query.Intent != IntentTraining {
		t.Errorf("expected Intent %s, got %s", IntentTraining, query.Intent)
	}
}

func TestQueryBuilder_WithOSVersion(t *testing.T) {
	query := NewQueryBuilder().
		WithOSVersion(OSUbuntu, "22.04").
		Build()

	if query.Os != OSUbuntu {
		t.Errorf("expected Os %s, got %s", OSUbuntu, query.Os)
	}

	expectedVersion, _ := version.ParseVersion("22.04")
	if query.OsVersion == nil || *query.OsVersion != expectedVersion {
		t.Errorf("expected OsVersion %v, got %v", expectedVersion, query.OsVersion)
	}
}

func TestQueryBuilder_WithKernel(t *testing.T) {
	query := NewQueryBuilder().
		WithKernel("5.15.0").
		Build()

	expectedVersion, _ := version.ParseVersion("5.15.0")
	if query.Kernel == nil || *query.Kernel != expectedVersion {
		t.Errorf("expected Kernel %v, got %v", expectedVersion, query.Kernel)
	}
}

func TestQueryBuilder_WithK8s(t *testing.T) {
	query := NewQueryBuilder().
		WithK8s("1.28.3").
		Build()

	expectedVersion, _ := version.ParseVersion("1.28.3")
	if query.K8s == nil || *query.K8s != expectedVersion {
		t.Errorf("expected K8s %v, got %v", expectedVersion, query.K8s)
	}
}

func TestQueryBuilder_WithContext(t *testing.T) {
	query := NewQueryBuilder().
		WithContext(true).
		Build()

	if !query.IncludeContext {
		t.Error("expected IncludeContext to be true")
	}
}

func TestQueryBuilder_Defaults(t *testing.T) {
	query := NewQueryBuilder().Build()

	if query.Os != OSAny {
		t.Errorf("expected default Os %s, got %s", OSAny, query.Os)
	}
	if query.Service != ServiceAny {
		t.Errorf("expected default Service %s, got %s", ServiceAny, query.Service)
	}
	if query.GPU != GPUAny {
		t.Errorf("expected default GPU %s, got %s", GPUAny, query.GPU)
	}
	if query.Intent != IntentAny {
		t.Errorf("expected default Intent %s, got %s", IntentAny, query.Intent)
	}
}

func TestQueryBuilder_Chaining(t *testing.T) {
	query := NewQueryBuilder().
		WithOS(OSUbuntu).
		WithOSVersion(OSUbuntu, "24.04").
		WithKernel("6.8.0").
		WithService(ServiceAKS).
		WithK8s("1.29.0").
		WithGPU(GPUB200).
		WithIntent(IntentInference).
		WithContext(true).
		Build()

	// Verify all fields are set
	if query.Os != OSUbuntu {
		t.Errorf("expected Os %s, got %s", OSUbuntu, query.Os)
	}

	osVersion, _ := version.ParseVersion("24.04")
	if query.OsVersion == nil || *query.OsVersion != osVersion {
		t.Errorf("expected OsVersion %v, got %v", osVersion, query.OsVersion)
	}

	kernel, _ := version.ParseVersion("6.8.0")
	if query.Kernel == nil || *query.Kernel != kernel {
		t.Errorf("expected Kernel %v, got %v", kernel, query.Kernel)
	}

	if query.Service != ServiceAKS {
		t.Errorf("expected Service %s, got %s", ServiceAKS, query.Service)
	}

	k8s, _ := version.ParseVersion("1.29.0")
	if query.K8s == nil || *query.K8s != k8s {
		t.Errorf("expected K8s %v, got %v", k8s, query.K8s)
	}

	if query.GPU != GPUB200 {
		t.Errorf("expected GPU %s, got %s", GPUB200, query.GPU)
	}

	if query.Intent != IntentInference {
		t.Errorf("expected Intent %s, got %s", IntentInference, query.Intent)
	}

	if !query.IncludeContext {
		t.Error("expected IncludeContext to be true")
	}
}

func TestQueryBuilder_InvalidVersionsIgnored(t *testing.T) {
	query := NewQueryBuilder().
		WithOSVersion(OSUbuntu, "invalid").
		WithKernel("bad-version").
		WithK8s("not-a-version").
		Build()

	// Invalid versions should be ignored, leaving nil pointers
	if query.OsVersion != nil {
		t.Errorf("expected OsVersion to be nil, got %v", query.OsVersion)
	}
	if query.Kernel != nil {
		t.Errorf("expected Kernel to be nil, got %v", query.Kernel)
	}
	if query.K8s != nil {
		t.Errorf("expected K8s to be nil, got %v", query.K8s)
	}
}

func TestQueryBuilder_BuildAndValidate_Valid(t *testing.T) {
	query, err := NewQueryBuilder().
		WithOS(OSUbuntu).
		WithService(ServiceEKS).
		WithGPU(GPUH100).
		WithIntent(IntentTraining).
		BuildAndValidate()

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if query == nil {
		t.Fatal("expected query to be non-nil")
	}
}

func TestQueryBuilder_BuildAndValidate_Invalid(t *testing.T) {
	// Create a builder and manually set an invalid field
	builder := NewQueryBuilder()
	builder.query.Os = OsFamily("invalid")

	_, err := builder.BuildAndValidate()
	if err == nil {
		t.Fatal("expected validation error for invalid os")
	}
}

func TestQuery_Validate(t *testing.T) {
	tests := []struct {
		name    string
		query   *Query
		wantErr bool
	}{
		{
			name: "valid query",
			query: &Query{
				Os:      OSUbuntu,
				Service: ServiceEKS,
				GPU:     GPUH100,
				Intent:  IntentTraining,
			},
			wantErr: false,
		},
		{
			name:    "empty query",
			query:   &Query{},
			wantErr: false,
		},
		{
			name:    "nil query",
			query:   nil,
			wantErr: true,
		},
		{
			name: "invalid os",
			query: &Query{
				Os: OsFamily("invalid"),
			},
			wantErr: true,
		},
		{
			name: "invalid service",
			query: &Query{
				Service: ServiceType("invalid"),
			},
			wantErr: true,
		},
		{
			name: "invalid gpu",
			query: &Query{
				GPU: GPUType("invalid"),
			},
			wantErr: true,
		},
		{
			name: "invalid intent",
			query: &Query{
				Intent: IntentType("invalid"),
			},
			wantErr: true,
		},
		{
			name: "wildcard values are valid",
			query: &Query{
				Os:      OSAny,
				Service: ServiceAny,
				GPU:     GPUAny,
				Intent:  IntentAny,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.query.Validate()
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error but got none")
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
			}
		})
	}
}
