package gpuoperator

import (
	"fmt"
	"time"

	common "github.com/NVIDIA/cloud-native-stack/pkg/bundler/internal"
	"github.com/NVIDIA/cloud-native-stack/pkg/measurement"
	"github.com/NVIDIA/cloud-native-stack/pkg/recipe"
)

const (
	strTrue = "true"
)

// HelmValues represents the data structure for GPU Operator Helm values.
type HelmValues struct {
	Timestamp                     string
	DriverRegistry                common.ValueWithContext
	GPUOperatorVersion            common.ValueWithContext
	EnableDriver                  common.ValueWithContext
	DriverVersion                 common.ValueWithContext
	UseOpenKernelModule           common.ValueWithContext
	NvidiaContainerToolkitVersion common.ValueWithContext
	DevicePluginVersion           common.ValueWithContext
	DCGMVersion                   common.ValueWithContext
	DCGMExporterVersion           common.ValueWithContext
	MIGStrategy                   common.ValueWithContext
	EnableGDS                     common.ValueWithContext
	VGPULicenseServer             common.ValueWithContext
	EnableSecureBoot              common.ValueWithContext
	CustomLabels                  map[string]string
	Namespace                     string
	Version                       string
	RecipeVersion                 string
}

// GenerateHelmValues generates Helm values from a recipe.
func GenerateHelmValues(recipe *recipe.Recipe, config map[string]string) *HelmValues {
	values := &HelmValues{
		Timestamp:        time.Now().UTC().Format(time.RFC3339),
		DriverRegistry:   common.ValueWithContext{Value: common.GetConfigValue(config, "driver_registry", "nvcr.io/nvidia")},
		EnableDriver:     common.ValueWithContext{Value: true},
		MIGStrategy:      common.ValueWithContext{Value: "single"},
		EnableGDS:        common.ValueWithContext{Value: false},
		EnableSecureBoot: common.ValueWithContext{Value: false},
		CustomLabels:     common.ExtractCustomLabels(config),
		Namespace:        common.GetConfigValue(config, "namespace", Name),
		Version:          common.GetBundlerVersion(config),
		RecipeVersion:    common.GetRecipeBundlerVersion(recipe.Metadata),
	}

	// Extract GPU Operator configuration from recipe measurements
	for _, m := range recipe.Measurements {
		switch m.Type {
		case measurement.TypeK8s:
			values.extractK8sSettings(m)
		case measurement.TypeGPU:
			values.extractGPUSettings(m)
		case measurement.TypeSystemD, measurement.TypeOS:
			// Not used for Helm values generation
		}
	}

	// Apply config overrides
	values.applyConfigOverrides(config)

	return values
}

// extractK8sSettings extracts Kubernetes-related settings from measurements.
func (v *HelmValues) extractK8sSettings(m *measurement.Measurement) {
	for _, st := range m.Subtypes {
		// Extract context for this subtype
		subtypeContext := common.GetSubtypeContext(st.Context)

		// Extract version information from 'image' subtype
		if st.Name == "image" {
			if val, ok := st.Data["gpu-operator"]; ok {
				if s, ok := val.Any().(string); ok {
					ctx := common.GetFieldContext(st.Context, "gpu-operator", subtypeContext)
					v.GPUOperatorVersion = common.ValueWithContext{Value: s, Context: ctx}
				}
			}
			if val, ok := st.Data["driver"]; ok {
				if s, ok := val.Any().(string); ok {
					ctx := common.GetFieldContext(st.Context, "driver", subtypeContext)
					v.DriverVersion = common.ValueWithContext{Value: s, Context: ctx}
				}
			}
			if val, ok := st.Data["container-toolkit"]; ok {
				if s, ok := val.Any().(string); ok {
					ctx := common.GetFieldContext(st.Context, "container-toolkit", subtypeContext)
					v.NvidiaContainerToolkitVersion = common.ValueWithContext{Value: s, Context: ctx}
				}
			}
			if val, ok := st.Data["k8s-device-plugin"]; ok {
				if s, ok := val.Any().(string); ok {
					ctx := common.GetFieldContext(st.Context, "k8s-device-plugin", subtypeContext)
					v.DevicePluginVersion = common.ValueWithContext{Value: s, Context: ctx}
				}
			}
			if val, ok := st.Data["dcgm"]; ok {
				if s, ok := val.Any().(string); ok {
					ctx := common.GetFieldContext(st.Context, "dcgm", subtypeContext)
					v.DCGMVersion = common.ValueWithContext{Value: s, Context: ctx}
				}
			}
			if val, ok := st.Data["dcgm-exporter"]; ok {
				if s, ok := val.Any().(string); ok {
					ctx := common.GetFieldContext(st.Context, "dcgm-exporter", subtypeContext)
					v.DCGMExporterVersion = common.ValueWithContext{Value: s, Context: ctx}
				}
			}
		}

		// Extract configuration flags from 'config' subtype
		if st.Name == "config" {
			// MIG configuration (boolean in recipe)
			if val, ok := st.Data["mig"]; ok {
				if b, ok := val.Any().(bool); ok && b {
					ctx := common.GetFieldContext(st.Context, "mig", subtypeContext)
					v.MIGStrategy = common.ValueWithContext{Value: "mixed", Context: ctx}
				}
			}
			// UseOpenKernelModule (camelCase in recipe)
			if val, ok := st.Data["useOpenKernelModule"]; ok {
				if b, ok := val.Any().(bool); ok {
					ctx := common.GetFieldContext(st.Context, "useOpenKernelModule", subtypeContext)
					v.UseOpenKernelModule = common.ValueWithContext{Value: b, Context: ctx}
				}
			}
			// RDMA support (affects GDS)
			if val, ok := st.Data["rdma"]; ok {
				if b, ok := val.Any().(bool); ok {
					ctx := common.GetFieldContext(st.Context, "rdma", subtypeContext)
					v.EnableGDS = common.ValueWithContext{Value: b, Context: ctx}
				}
			}
		}
	}
}

// extractGPUSettings extracts GPU-related settings from measurements.
func (v *HelmValues) extractGPUSettings(m *measurement.Measurement) {
	for _, st := range m.Subtypes {
		subtypeContext := common.GetSubtypeContext(st.Context)

		// Recipe uses 'smi' subtype for nvidia-smi output
		if st.Name == "smi" {
			if val, ok := st.Data["driver-version"]; ok {
				if s, ok := val.Any().(string); ok {
					// Only set if not already set from K8s measurements
					if cv, ok := v.DriverVersion.Value.(string); !ok || cv == "" {
						ctx := common.GetFieldContext(st.Context, "driver-version", subtypeContext)
						v.DriverVersion = common.ValueWithContext{Value: s, Context: ctx}
					}
				}
			}
		}
	}
}

// applyConfigOverrides applies configuration overrides to values.
func (v *HelmValues) applyConfigOverrides(config map[string]string) {
	if val, ok := config["driver_version"]; ok && val != "" {
		v.DriverVersion = common.ValueWithContext{Value: val, Context: "Override from bundler configuration"}
	}
	if val, ok := config["gpu_operator_version"]; ok && val != "" {
		v.GPUOperatorVersion = common.ValueWithContext{Value: val, Context: "Override from bundler configuration"}
	}
	if val, ok := config["mig_strategy"]; ok && val != "" {
		v.MIGStrategy = common.ValueWithContext{Value: val, Context: "Override from bundler configuration"}
	}
	if val, ok := config["enable_gds"]; ok {
		v.EnableGDS = common.ValueWithContext{Value: val == strTrue, Context: "Override from bundler configuration"}
	}
	if val, ok := config["vgpu_license_server"]; ok && val != "" {
		v.VGPULicenseServer = common.ValueWithContext{Value: val, Context: "Override from bundler configuration"}
	}
	if val, ok := config["namespace"]; ok && val != "" {
		v.Namespace = val
	}

	// Custom labels
	for k, val := range config {
		if len(k) > 6 && k[:6] == "label_" {
			v.CustomLabels[k[6:]] = val
		}
	}
}

// ToMap converts HelmValues to a map for template rendering.
func (v *HelmValues) ToMap() map[string]interface{} {
	return map[string]interface{}{
		"Timestamp":                     v.Timestamp,
		"DriverRegistry":                v.DriverRegistry,
		"GPUOperatorVersion":            v.GPUOperatorVersion,
		"EnableDriver":                  v.EnableDriver,
		"DriverVersion":                 v.DriverVersion,
		"UseOpenKernelModule":           v.UseOpenKernelModule,
		"NvidiaContainerToolkitVersion": v.NvidiaContainerToolkitVersion,
		"DevicePluginVersion":           v.DevicePluginVersion,
		"DCGMVersion":                   v.DCGMVersion,
		"DCGMExporterVersion":           v.DCGMExporterVersion,
		"MIGStrategy":                   v.MIGStrategy,
		"EnableGDS":                     v.EnableGDS,
		"VGPULicenseServer":             v.VGPULicenseServer,
		"EnableSecureBoot":              v.EnableSecureBoot,
		"CustomLabels":                  v.CustomLabels,
		"Namespace":                     v.Namespace,
		"Version":                       v.Version,
		"RecipeVersion":                 v.RecipeVersion,
	}
}

// Validate validates the Helm values.
func (v *HelmValues) Validate() error {
	if v.Namespace == "" {
		return fmt.Errorf("namespace cannot be empty")
	}
	if dr, ok := v.DriverRegistry.Value.(string); !ok || dr == "" {
		return fmt.Errorf("driver registry cannot be empty")
	}
	if ms, ok := v.MIGStrategy.Value.(string); ok {
		if ms != "single" && ms != "mixed" {
			return fmt.Errorf("invalid MIG strategy: %s (must be single or mixed)", ms)
		}
	}
	return nil
}
