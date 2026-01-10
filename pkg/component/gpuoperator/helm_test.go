/*
Copyright © 2025 NVIDIA Corporation
SPDX-License-Identifier: Apache-2.0
*/

package gpuoperator

import (
	"testing"

	"github.com/NVIDIA/cloud-native-stack/pkg/measurement"
	"github.com/NVIDIA/cloud-native-stack/pkg/recipe"
	"github.com/NVIDIA/cloud-native-stack/pkg/recipe/header"
)

const (
	testValueTrue   = "true"
	testValueFalse  = "false"
	testValueSingle = "single"
	testValueMixed  = "mixed"
)

func TestGenerateHelmValues_WithValueOverrides(t *testing.T) {
	tests := []struct {
		name      string
		recipe    *recipe.Recipe
		config    map[string]string
		overrides map[string]string
		verify    func(t *testing.T, values *HelmValues)
	}{
		{
			name:   "no overrides",
			recipe: createMinimalRecipe(),
			config: map[string]string{
				"namespace": "gpu-operator",
			},
			overrides: nil,
			verify: func(t *testing.T, values *HelmValues) {
				if values.Namespace != "gpu-operator" {
					t.Errorf("expected namespace 'gpu-operator', got %q", values.Namespace)
				}
				if values.EnableGDS != testValueFalse {
					t.Errorf("expected EnableGDS %q, got %q", testValueFalse, values.EnableGDS)
				}
				if values.MIGStrategy != testValueSingle {
					t.Errorf("expected MIGStrategy %q, got %q", testValueSingle, values.MIGStrategy)
				}
			},
		},
		{
			name:   "single override - gds.enabled",
			recipe: createMinimalRecipe(),
			config: map[string]string{},
			overrides: map[string]string{
				"gds.enabled": testValueTrue,
			},
			verify: func(t *testing.T, values *HelmValues) {
				if values.EnableGDS != testValueTrue {
					t.Errorf("expected EnableGDS %q, got %q", testValueTrue, values.EnableGDS)
				}
			},
		},
		{
			name:   "multiple overrides same bundler",
			recipe: createMinimalRecipe(),
			config: map[string]string{},
			overrides: map[string]string{
				"gds.enabled":  testValueTrue,
				"mig.strategy": testValueMixed,
			},
			verify: func(t *testing.T, values *HelmValues) {
				if values.EnableGDS != testValueTrue {
					t.Errorf("expected EnableGDS %q, got %q", testValueTrue, values.EnableGDS)
				}
				if values.MIGStrategy != testValueMixed {
					t.Errorf("expected MIGStrategy %q, got %q", testValueMixed, values.MIGStrategy)
				}
			},
		},
		{
			name:   "driver version override",
			recipe: createMinimalRecipe(),
			config: map[string]string{
				"driver_version": "570.86.16",
			},
			overrides: map[string]string{
				"driver.version": "580.82.07",
			},
			verify: func(t *testing.T, values *HelmValues) {
				// Override should take precedence over config
				if values.DriverVersion != "580.82.07" {
					t.Errorf("expected DriverVersion '580.82.07', got %q", values.DriverVersion)
				}
			},
		},
		{
			name:   "namespace override",
			recipe: createMinimalRecipe(),
			config: map[string]string{
				"namespace": "default",
			},
			overrides: map[string]string{
				"namespace": "custom-namespace",
			},
			verify: func(t *testing.T, values *HelmValues) {
				if values.Namespace != "custom-namespace" {
					t.Errorf("expected namespace 'custom-namespace', got %q", values.Namespace)
				}
			},
		},
		{
			name:   "operator version override",
			recipe: createMinimalRecipe(),
			config: map[string]string{},
			overrides: map[string]string{
				"operator.version": "v25.4.0",
			},
			verify: func(t *testing.T, values *HelmValues) {
				if values.GPUOperatorVersion != "v25.4.0" {
					t.Errorf("expected GPUOperatorVersion 'v25.4.0', got %q", values.GPUOperatorVersion)
				}
			},
		},
		{
			name:   "toolkit version override",
			recipe: createMinimalRecipe(),
			config: map[string]string{},
			overrides: map[string]string{
				"toolkit.version": "v1.18.0",
			},
			verify: func(t *testing.T, values *HelmValues) {
				if values.NvidiaContainerToolkitVersion != "v1.18.0" {
					t.Errorf("expected toolkit version 'v1.18.0', got %q", values.NvidiaContainerToolkitVersion)
				}
			},
		},
		{
			name:   "boolean value parsing",
			recipe: createMinimalRecipe(),
			config: map[string]string{},
			overrides: map[string]string{
				"driver.enabled": testValueFalse,
			},
			verify: func(t *testing.T, values *HelmValues) {
				if values.EnableDriver != testValueFalse {
					t.Errorf("expected EnableDriver %q, got %q", testValueFalse, values.EnableDriver)
				}
			},
		},
		{
			name:   "custom node selector labels",
			recipe: createMinimalRecipe(),
			config: map[string]string{},
			overrides: map[string]string{
				"nodeSelector.accelerator": "nvidia-h100",
			},
			verify: func(t *testing.T, values *HelmValues) {
				if values.CustomLabels == nil {
					t.Fatal("expected CustomLabels to be set")
				}
				if values.CustomLabels["accelerator"] != "nvidia-h100" {
					t.Errorf("expected custom label 'accelerator' = 'nvidia-h100', got %q",
						values.CustomLabels["accelerator"])
				}
			},
		},
		{
			name:   "driver repository override",
			recipe: createMinimalRecipe(),
			config: map[string]string{
				"driver_registry": "nvcr.io/nvidia",
			},
			overrides: map[string]string{
				"driver.repository": "custom-registry.io/drivers",
			},
			verify: func(t *testing.T, values *HelmValues) {
				if values.DriverRegistry != "custom-registry.io/drivers" {
					t.Errorf("expected DriverRegistry 'custom-registry.io/drivers', got %q",
						values.DriverRegistry)
				}
			},
		},
		{
			name:   "multiple overrides comprehensive test",
			recipe: createMinimalRecipe(),
			config: map[string]string{
				"namespace": "gpu-operator",
			},
			overrides: map[string]string{
				"gds.enabled":                 testValueTrue,
				"mig.strategy":                testValueMixed,
				"driver.version":              "580.82.07",
				"operator.version":            "v25.4.0",
				"nodeSelector.gpu":            "nvidia",
				"nodeSelector.accelerator":    "h100",
				"sandboxWorkloads.enabled":    testValueTrue,
				"driver.useOpenKernelModules": testValueTrue,
			},
			verify: func(t *testing.T, values *HelmValues) {
				if values.EnableGDS != testValueTrue {
					t.Errorf("expected EnableGDS %q, got %q", testValueTrue, values.EnableGDS)
				}
				if values.MIGStrategy != testValueMixed {
					t.Errorf("expected MIGStrategy %q, got %q", testValueMixed, values.MIGStrategy)
				}
				if values.DriverVersion != "580.82.07" {
					t.Errorf("expected DriverVersion '580.82.07', got %q", values.DriverVersion)
				}
				if values.GPUOperatorVersion != "v25.4.0" {
					t.Errorf("expected GPUOperatorVersion 'v25.4.0', got %q", values.GPUOperatorVersion)
				}
				if values.EnableSecureBoot != testValueTrue {
					t.Errorf("expected EnableSecureBoot %q, got %q", testValueTrue, values.EnableSecureBoot)
				}
				if values.UseOpenKernelModule != testValueTrue {
					t.Errorf("expected UseOpenKernelModule %q, got %q", testValueTrue, values.UseOpenKernelModule)
				}
				if values.CustomLabels == nil {
					t.Fatal("expected CustomLabels to be set")
				}
				if values.CustomLabels["gpu"] != "nvidia" {
					t.Errorf("expected custom label 'gpu' = 'nvidia', got %q",
						values.CustomLabels["gpu"])
				}
				if values.CustomLabels["accelerator"] != "h100" {
					t.Errorf("expected custom label 'accelerator' = 'h100', got %q",
						values.CustomLabels["accelerator"])
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			values := GenerateHelmValues(tt.recipe, tt.config, tt.overrides)
			tt.verify(t, values)
		})
	}
}

// createMinimalRecipe creates a minimal recipe for testing.
func createMinimalRecipe() *recipe.Recipe {
	return &recipe.Recipe{
		Header: header.Header{
			APIVersion: "v1",
			Kind:       header.KindRecipe,
		},
		Measurements: []*measurement.Measurement{
			{
				Type: measurement.TypeK8s,
				Subtypes: []measurement.Subtype{
					{
						Name: "image",
						Data: map[string]measurement.Reading{
							"gpu-operator": measurement.Str("v25.3.3"),
							"driver":       measurement.Str("570.86.16"),
						},
					},
				},
			},
		},
	}
}
