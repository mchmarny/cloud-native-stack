package internal

import (
	"testing"
)

// TestStruct is a test struct with various field types.
type TestStruct struct {
	// Simple fields
	Name    string
	Enabled string
	Count   int

	// Nested struct
	Driver struct {
		Version string
		Enabled string
	}

	// Acronym fields
	EnableGDS string
	MIG       struct {
		Strategy string
	}
	GPUOperator struct {
		Version string
	}

	// Complex nested
	DCGM struct {
		Exporter struct {
			Version string
			Enabled string
		}
	}
}

func TestApplyValueOverrides_SimpleFields(t *testing.T) {
	tests := []struct {
		name      string
		overrides map[string]string
		want      TestStruct
		wantErr   bool
	}{
		{
			name: "set string field",
			overrides: map[string]string{
				"name": "test-value",
			},
			want: TestStruct{
				Name: "test-value",
			},
		},
		{
			name: "set enabled field",
			overrides: map[string]string{
				"enabled": "true",
			},
			want: TestStruct{
				Enabled: "true",
			},
		},
		{
			name: "set multiple fields",
			overrides: map[string]string{
				"name":    "test",
				"enabled": "false",
			},
			want: TestStruct{
				Name:    "test",
				Enabled: "false",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := TestStruct{}
			err := ApplyValueOverrides(&got, tt.overrides)

			if (err != nil) != tt.wantErr {
				t.Errorf("ApplyValueOverrides() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if got.Name != tt.want.Name {
				t.Errorf("Name = %v, want %v", got.Name, tt.want.Name)
			}
			if got.Enabled != tt.want.Enabled {
				t.Errorf("Enabled = %v, want %v", got.Enabled, tt.want.Enabled)
			}
		})
	}
}

func TestApplyValueOverrides_NestedFields(t *testing.T) {
	tests := []struct {
		name      string
		overrides map[string]string
		want      TestStruct
		wantErr   bool
	}{
		{
			name: "set nested field",
			overrides: map[string]string{
				"driver.version": "550.127",
			},
			want: TestStruct{
				Driver: struct {
					Version string
					Enabled string
				}{
					Version: "550.127",
				},
			},
		},
		{
			name: "set multiple nested fields",
			overrides: map[string]string{
				"driver.version": "550.127",
				"driver.enabled": "true",
			},
			want: TestStruct{
				Driver: struct {
					Version string
					Enabled string
				}{
					Version: "550.127",
					Enabled: "true",
				},
			},
		},
		{
			name: "set deeply nested field",
			overrides: map[string]string{
				"dcgm.exporter.version": "3.3.11",
				"dcgm.exporter.enabled": "true",
			},
			want: TestStruct{
				DCGM: struct {
					Exporter struct {
						Version string
						Enabled string
					}
				}{
					Exporter: struct {
						Version string
						Enabled string
					}{
						Version: "3.3.11",
						Enabled: "true",
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := TestStruct{}
			err := ApplyValueOverrides(&got, tt.overrides)

			if (err != nil) != tt.wantErr {
				t.Errorf("ApplyValueOverrides() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.want.Driver.Version != "" && got.Driver.Version != tt.want.Driver.Version {
				t.Errorf("Driver.Version = %v, want %v", got.Driver.Version, tt.want.Driver.Version)
			}
			if tt.want.Driver.Enabled != "" && got.Driver.Enabled != tt.want.Driver.Enabled {
				t.Errorf("Driver.Enabled = %v, want %v", got.Driver.Enabled, tt.want.Driver.Enabled)
			}
			if tt.want.DCGM.Exporter.Version != "" && got.DCGM.Exporter.Version != tt.want.DCGM.Exporter.Version {
				t.Errorf("DCGM.Exporter.Version = %v, want %v", got.DCGM.Exporter.Version, tt.want.DCGM.Exporter.Version)
			}
		})
	}
}

func TestApplyValueOverrides_AcronymFields(t *testing.T) {
	tests := []struct {
		name      string
		overrides map[string]string
		want      TestStruct
		wantErr   bool
	}{
		{
			name: "set MIG strategy",
			overrides: map[string]string{
				"mig.strategy": "mixed",
			},
			want: TestStruct{
				MIG: struct {
					Strategy string
				}{
					Strategy: "mixed",
				},
			},
		},
		{
			name: "set GPU operator version",
			overrides: map[string]string{
				"gpu-operator.version": "25.3.3",
			},
			want: TestStruct{
				GPUOperator: struct {
					Version string
				}{
					Version: "25.3.3",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := TestStruct{}
			err := ApplyValueOverrides(&got, tt.overrides)

			if (err != nil) != tt.wantErr {
				t.Errorf("ApplyValueOverrides() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.want.MIG.Strategy != "" && got.MIG.Strategy != tt.want.MIG.Strategy {
				t.Errorf("MIG.Strategy = %v, want %v", got.MIG.Strategy, tt.want.MIG.Strategy)
			}
			if tt.want.GPUOperator.Version != "" && got.GPUOperator.Version != tt.want.GPUOperator.Version {
				t.Errorf("GPUOperator.Version = %v, want %v", got.GPUOperator.Version, tt.want.GPUOperator.Version)
			}
		})
	}
}

func TestApplyValueOverrides_Errors(t *testing.T) {
	tests := []struct {
		name      string
		target    interface{}
		overrides map[string]string
		wantErr   bool
		errMsg    string
	}{
		{
			name:   "non-pointer target",
			target: TestStruct{},
			overrides: map[string]string{
				"name": "test",
			},
			wantErr: true,
			errMsg:  "must be a pointer",
		},
		{
			name:      "nil overrides",
			target:    &TestStruct{},
			overrides: nil,
			wantErr:   false,
		},
		{
			name:      "empty overrides",
			target:    &TestStruct{},
			overrides: map[string]string{},
			wantErr:   false,
		},
		{
			name:   "non-existent field",
			target: &TestStruct{},
			overrides: map[string]string{
				"nonexistent": "value",
			},
			wantErr: true,
			errMsg:  "field not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ApplyValueOverrides(tt.target, tt.overrides)

			if (err != nil) != tt.wantErr {
				t.Errorf("ApplyValueOverrides() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.errMsg != "" {
				if err == nil || !containsSubstring(err.Error(), tt.errMsg) {
					t.Errorf("Expected error containing %q, got %v", tt.errMsg, err)
				}
			}
		})
	}
}

func TestApplyValueOverrides_CaseInsensitive(t *testing.T) {
	tests := []struct {
		name      string
		overrides map[string]string
		want      TestStruct
	}{
		{
			name: "lowercase field name",
			overrides: map[string]string{
				"name": "test",
			},
			want: TestStruct{
				Name: "test",
			},
		},
		{
			name: "uppercase field name",
			overrides: map[string]string{
				"NAME": "test",
			},
			want: TestStruct{
				Name: "test",
			},
		},
		{
			name: "mixed case field name",
			overrides: map[string]string{
				"NaMe": "test",
			},
			want: TestStruct{
				Name: "test",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := TestStruct{}
			err := ApplyValueOverrides(&got, tt.overrides)

			if err != nil {
				t.Errorf("ApplyValueOverrides() unexpected error = %v", err)
				return
			}

			if got.Name != tt.want.Name {
				t.Errorf("Name = %v, want %v", got.Name, tt.want.Name)
			}
		})
	}
}

// Test with actual GPU Operator-like struct
type GPUOperatorValues struct {
	EnableDriver string
	Driver       struct {
		Version string
		Enabled string
	}
	EnableGDS string
	GDS       struct {
		Enabled string
	}
	GDRCopy struct {
		Enabled string
	}
	MIG struct {
		Strategy string
	}
	DCGM struct {
		Version string
	}
}

func TestApplyValueOverrides_GPUOperatorScenarios(t *testing.T) {
	tests := []struct {
		name      string
		overrides map[string]string
		verify    func(t *testing.T, values *GPUOperatorValues)
	}{
		{
			name: "gdrcopy enabled override",
			overrides: map[string]string{
				"gdrcopy.enabled": "false",
			},
			verify: func(t *testing.T, values *GPUOperatorValues) {
				if values.GDRCopy.Enabled != "false" {
					t.Errorf("GDRCopy.Enabled = %v, want false", values.GDRCopy.Enabled)
				}
			},
		},
		{
			name: "gds enabled override",
			overrides: map[string]string{
				"gds.enabled": "true",
			},
			verify: func(t *testing.T, values *GPUOperatorValues) {
				// Should match either EnableGDS or GDS.Enabled
				if values.EnableGDS != "true" && values.GDS.Enabled != "true" {
					t.Errorf("GDS not enabled: EnableGDS=%v, GDS.Enabled=%v", values.EnableGDS, values.GDS.Enabled)
				}
			},
		},
		{
			name: "driver version override",
			overrides: map[string]string{
				"driver.version": "570.86.16",
			},
			verify: func(t *testing.T, values *GPUOperatorValues) {
				if values.Driver.Version != "570.86.16" {
					t.Errorf("Driver.Version = %v, want 570.86.16", values.Driver.Version)
				}
			},
		},
		{
			name: "mig strategy override",
			overrides: map[string]string{
				"mig.strategy": "mixed",
			},
			verify: func(t *testing.T, values *GPUOperatorValues) {
				if values.MIG.Strategy != "mixed" {
					t.Errorf("MIG.Strategy = %v, want mixed", values.MIG.Strategy)
				}
			},
		},
		{
			name: "multiple overrides",
			overrides: map[string]string{
				"gdrcopy.enabled": "false",
				"gds.enabled":     "true",
				"driver.version":  "570.86.16",
				"mig.strategy":    "mixed",
			},
			verify: func(t *testing.T, values *GPUOperatorValues) {
				if values.GDRCopy.Enabled != "false" {
					t.Errorf("GDRCopy.Enabled = %v, want false", values.GDRCopy.Enabled)
				}
				if values.Driver.Version != "570.86.16" {
					t.Errorf("Driver.Version = %v, want 570.86.16", values.Driver.Version)
				}
				if values.MIG.Strategy != "mixed" {
					t.Errorf("MIG.Strategy = %v, want mixed", values.MIG.Strategy)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			values := &GPUOperatorValues{}
			err := ApplyValueOverrides(values, tt.overrides)

			if err != nil {
				t.Fatalf("ApplyValueOverrides() unexpected error = %v", err)
			}

			tt.verify(t, values)
		})
	}
}

// containsSubstring checks if string contains substring (renamed to avoid collision)
func containsSubstring(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && containsSubstringHelper(s, substr))
}

func containsSubstringHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
