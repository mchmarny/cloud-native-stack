package config

import (
	"testing"
)

const (
	testLabelValue      = "label"
	testAnnotationValue = "annotation"
	testValue1          = "value1"
)

func TestNewConfig(t *testing.T) {
	cfg := NewConfig()

	if cfg == nil {
		t.Fatal("NewConfig() returned nil")
		return
	}

	// Check defaults
	if cfg.OutputFormat != "yaml" {
		t.Errorf("OutputFormat = %s, want yaml", cfg.OutputFormat)
	}

	if cfg.Compression {
		t.Error("Compression should be false by default")
	}

	if !cfg.IncludeScripts {
		t.Error("IncludeScripts should be true by default")
	}

	if !cfg.IncludeReadme {
		t.Error("IncludeReadme should be true by default")
	}

	if !cfg.IncludeChecksums {
		t.Error("IncludeChecksums should be true by default")
	}

	if cfg.Namespace != "default" {
		t.Errorf("Namespace = %s, want default", cfg.Namespace)
	}

	if cfg.Verbose {
		t.Error("Verbose should be false by default")
	}

	if cfg.CustomLabels == nil {
		t.Error("CustomLabels should be initialized")
	}

	if cfg.CustomAnnotations == nil {
		t.Error("CustomAnnotations should be initialized")
	}
}

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name: "valid yaml format",
			config: &Config{
				OutputFormat: "yaml",
				Namespace:    "gpu-operator",
			},
			wantErr: false,
		},
		{
			name: "valid json format",
			config: &Config{
				OutputFormat: "json",
				Namespace:    "gpu-operator",
			},
			wantErr: false,
		},
		{
			name: "valid helm format",
			config: &Config{
				OutputFormat: "helm",
				Namespace:    "gpu-operator",
			},
			wantErr: false,
		},
		{
			name: "invalid format",
			config: &Config{
				OutputFormat: "xml",
				Namespace:    "gpu-operator",
			},
			wantErr: true,
		},
		{
			name: "empty namespace",
			config: &Config{
				OutputFormat: "yaml",
				Namespace:    "",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestConfig_Merge(t *testing.T) {
	t.Run("merge with nil", func(t *testing.T) {
		cfg := NewConfig()
		original := cfg.OutputFormat

		cfg.Merge(nil)

		if cfg.OutputFormat != original {
			t.Error("Merge(nil) should not modify config")
		}
	})

	t.Run("merge overlapping fields", func(t *testing.T) {
		cfg := NewConfig()
		cfg.OutputFormat = "yaml"
		cfg.Namespace = "default"
		cfg.Compression = false

		other := &Config{
			OutputFormat: "json",
			Namespace:    "gpu-operator",
			Compression:  true,
		}

		cfg.Merge(other)

		if cfg.OutputFormat != "json" {
			t.Errorf("OutputFormat = %s, want json", cfg.OutputFormat)
		}

		if cfg.Namespace != "gpu-operator" {
			t.Errorf("Namespace = %s, want gpu-operator", cfg.Namespace)
		}

		if !cfg.Compression {
			t.Error("Compression should be true after merge")
		}
	})

	t.Run("merge labels and annotations", func(t *testing.T) {
		cfg := NewConfig()
		cfg.CustomLabels["existing"] = testLabelValue
		cfg.CustomAnnotations["existing"] = testAnnotationValue

		other := &Config{
			CustomLabels:      map[string]string{"new": testLabelValue},
			CustomAnnotations: map[string]string{"new": testAnnotationValue},
		}

		cfg.Merge(other)

		if cfg.CustomLabels["existing"] != testLabelValue {
			t.Error("Merge should preserve existing labels")
		}

		if cfg.CustomLabels["new"] != testLabelValue {
			t.Error("Merge should add new labels")
		}

		if cfg.CustomAnnotations["existing"] != testAnnotationValue {
			t.Error("Merge should preserve existing annotations")
		}

		if cfg.CustomAnnotations["new"] != testAnnotationValue {
			t.Error("Merge should add new annotations")
		}
	})

	t.Run("merge helm configuration", func(t *testing.T) {
		cfg := NewConfig()

		other := &Config{
			HelmChartVersion: "1.2.3",
			HelmRepository:   "https://example.com/helm",
		}

		cfg.Merge(other)

		if cfg.HelmChartVersion != "1.2.3" {
			t.Errorf("HelmChartVersion = %s, want 1.2.3", cfg.HelmChartVersion)
		}

		if cfg.HelmRepository != "https://example.com/helm" {
			t.Errorf("HelmRepository = %s, want https://example.com/helm", cfg.HelmRepository)
		}
	})

	t.Run("merge boolean flags", func(t *testing.T) {
		cfg := NewConfig()
		cfg.Verbose = false

		other := &Config{
			Verbose:          true,
			IncludeScripts:   false,
			IncludeReadme:    false,
			IncludeChecksums: false,
		}

		cfg.Merge(other)

		if !cfg.Verbose {
			t.Error("Verbose should be true after merge")
		}

		if cfg.IncludeScripts {
			t.Error("IncludeScripts should be false after merge")
		}

		if cfg.IncludeReadme {
			t.Error("IncludeReadme should be false after merge")
		}

		if cfg.IncludeChecksums {
			t.Error("IncludeChecksums should be false after merge")
		}
	})
}

func TestConfig_Clone(t *testing.T) {
	original := NewConfig()
	original.OutputFormat = "json"
	original.Namespace = "test-namespace"
	original.Compression = true
	original.CustomLabels["key1"] = testValue1
	original.CustomAnnotations["anno1"] = testValue1

	clone := original.Clone()

	// Verify clone has same values
	if clone.OutputFormat != original.OutputFormat {
		t.Errorf("Clone OutputFormat = %s, want %s", clone.OutputFormat, original.OutputFormat)
	}

	if clone.Namespace != original.Namespace {
		t.Errorf("Clone Namespace = %s, want %s", clone.Namespace, original.Namespace)
	}

	if clone.Compression != original.Compression {
		t.Error("Clone Compression should match original")
	}

	// Verify maps are deep copied
	if clone.CustomLabels["key1"] != testValue1 {
		t.Error("Clone should have same label values")
	}

	// Modify clone and ensure original is unaffected
	clone.CustomLabels["key2"] = "value2"
	clone.CustomAnnotations["anno2"] = "value2"

	if _, exists := original.CustomLabels["key2"]; exists {
		t.Error("Modifying clone should not affect original labels")
	}

	if _, exists := original.CustomAnnotations["anno2"]; exists {
		t.Error("Modifying clone should not affect original annotations")
	}

	// Verify clone maps are independent
	if len(clone.CustomLabels) == len(original.CustomLabels) {
		t.Error("Clone maps should be independent")
	}
}
