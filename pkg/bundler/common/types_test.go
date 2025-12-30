package common

import (
	"testing"
)

func TestParseType(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    BundleType
		wantErr bool
	}{
		{
			name:    "gpu-operator",
			input:   "gpu-operator",
			want:    BundleTypeGpuOperator,
			wantErr: false,
		},
		{
			name:    "network-operator",
			input:   "network-operator",
			want:    BundleTypeNetworkOperator,
			wantErr: false,
		},
		{
			name:    "unknown type",
			input:   "unknown-operator",
			want:    "",
			wantErr: true,
		},
		{
			name:    "empty string",
			input:   "",
			want:    "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseType(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseType() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ParseType() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSupportedTypes(t *testing.T) {
	types := SupportedTypes()

	if len(types) != 2 {
		t.Errorf("SupportedTypes() returned %d types, want 2", len(types))
	}

	// Check that both expected types are present
	foundGpu := false
	foundNetwork := false

	for _, typ := range types {
		if typ == BundleTypeGpuOperator {
			foundGpu = true
		}
		if typ == BundleTypeNetworkOperator {
			foundNetwork = true
		}
	}

	if !foundGpu {
		t.Error("SupportedTypes() missing gpu-operator")
	}
	if !foundNetwork {
		t.Error("SupportedTypes() missing network-operator")
	}
}

func TestSupportedBundleTypesAsStrings(t *testing.T) {
	strs := SupportedBundleTypesAsStrings()

	if len(strs) != 2 {
		t.Errorf("SupportedBundleTypesAsStrings() returned %d strings, want 2", len(strs))
	}

	// Check that strings match the types
	foundGpu := false
	foundNetwork := false

	for _, s := range strs {
		if s == "gpu-operator" {
			foundGpu = true
		}
		if s == "network-operator" {
			foundNetwork = true
		}
	}

	if !foundGpu {
		t.Error("SupportedBundleTypesAsStrings() missing 'gpu-operator'")
	}
	if !foundNetwork {
		t.Error("SupportedBundleTypesAsStrings() missing 'network-operator'")
	}
}

func TestType_String(t *testing.T) {
	tests := []struct {
		name string
		typ  BundleType
		want string
	}{
		{
			name: "gpu-operator",
			typ:  BundleTypeGpuOperator,
			want: "gpu-operator",
		},
		{
			name: "network-operator",
			typ:  BundleTypeNetworkOperator,
			want: "network-operator",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := string(tt.typ)
			if got != tt.want {
				t.Errorf("Type string = %v, want %v", got, tt.want)
			}
		})
	}
}
