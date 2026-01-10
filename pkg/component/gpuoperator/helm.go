package gpuoperator

import (
	"fmt"
	"time"

	common "github.com/NVIDIA/cloud-native-stack/pkg/component/internal"
	"github.com/NVIDIA/cloud-native-stack/pkg/measurement"
	"github.com/NVIDIA/cloud-native-stack/pkg/recipe"
)

// HelmValues represents the data structure for GPU Operator Helm values.
// All fields are strings to support template rendering with type conversion.
type HelmValues struct {
	Timestamp     string
	Version       string
	RecipeVersion string
	Namespace     string
	CustomLabels  map[string]string

	// Operator
	GPUOperatorVersion string
	DriverRegistry     string

	// Driver
	EnableDriver        string
	DriverVersion       string
	UseOpenKernelModule string

	// Container Toolkit
	NvidiaContainerToolkitVersion string

	// Device Plugin
	DevicePluginVersion string

	// DCGM
	DCGMVersion string

	// DCGM Exporter
	DCGMExporterVersion string

	// MIG Manager
	MIGStrategy string

	// GDS (GPUDirect Storage)
	EnableGDS string

	// GDRCopy (GPUDirect RDMA Copy)
	GDRCopy struct {
		Enabled string
	}

	// GFD (GPU Feature Discovery)
	GFD struct {
		Enabled string
	}

	// Node Status Exporter
	NodeStatusExporter struct {
		Enabled string
	}

	// NVIDIA Driver CRD
	NvidiaDriverCRD struct {
		Enabled string
	}

	// Validator
	Validator struct {
		Enabled string
	}

	// VFIO Manager
	VFIOManager struct {
		Enabled string
	}

	// vGPU Device Manager
	VGPUDeviceManager struct {
		Enabled string
	}

	// vGPU License Server
	VGPULicenseServer string

	// Sandbox Workloads / Secure Boot
	EnableSecureBoot string
}

// GenerateHelmValues generates Helm values from a recipe.
func GenerateHelmValues(recipe *recipe.Recipe, config map[string]string, overrides map[string]string) *HelmValues {
	values := &HelmValues{
		Timestamp:        time.Now().UTC().Format(time.RFC3339),
		DriverRegistry:   common.GetConfigValue(config, "driver_registry", "nvcr.io/nvidia"),
		EnableDriver:     "true",
		MIGStrategy:      "single",
		EnableGDS:        "false",
		EnableSecureBoot: "false",
		CustomLabels:     common.ExtractCustomLabels(config),
		Namespace:        common.GetConfigValue(config, "namespace", Name),
		Version:          common.GetBundlerVersion(config),
		RecipeVersion:    common.GetRecipeBundlerVersion(config),
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

	// Apply user value overrides from --set flags
	values.applyValueOverrides(overrides)

	return values
}

// GenerateHelmValuesFromMap generates Helm values from config map (for RecipeResult inputs).
func GenerateHelmValuesFromMap(config map[string]string) *HelmValues {
	helmValues := &HelmValues{
		Timestamp:        time.Now().UTC().Format(time.RFC3339),
		DriverRegistry:   common.GetConfigValue(config, "driver_registry", "nvcr.io/nvidia"),
		EnableDriver:     "true",
		MIGStrategy:      "single",
		EnableGDS:        "false",
		EnableSecureBoot: "false",
		Namespace:        common.GetConfigValue(config, "namespace", Name),
		Version:          common.GetBundlerVersion(config),
		RecipeVersion:    common.GetRecipeBundlerVersion(config),
	}

	return helmValues
}

// extractK8sSettings extracts Kubernetes-related settings from measurements.
func (v *HelmValues) extractK8sSettings(m *measurement.Measurement) {
	for _, st := range m.Subtypes {
		// Extract version information from 'image' subtype
		if st.Name == "image" {
			if val, ok := st.Data["gpu-operator"]; ok {
				if s, ok := val.Any().(string); ok {
					v.GPUOperatorVersion = s
				}
			}
			if val, ok := st.Data["driver"]; ok {
				if s, ok := val.Any().(string); ok {
					v.DriverVersion = s
				}
			}
			if val, ok := st.Data["container-toolkit"]; ok {
				if s, ok := val.Any().(string); ok {
					v.NvidiaContainerToolkitVersion = s
				}
			}
			if val, ok := st.Data["k8s-device-plugin"]; ok {
				if s, ok := val.Any().(string); ok {
					v.DevicePluginVersion = s
				}
			}
			if val, ok := st.Data["dcgm"]; ok {
				if s, ok := val.Any().(string); ok {
					v.DCGMVersion = s
				}
			}
			if val, ok := st.Data["dcgm-exporter"]; ok {
				if s, ok := val.Any().(string); ok {
					v.DCGMExporterVersion = s
				}
			}
		}

		// Extract configuration flags from 'config' subtype
		if st.Name == "config" {
			// MIG configuration (boolean in recipe)
			if val, ok := st.Data["mig"]; ok {
				if b, ok := val.Any().(bool); ok && b {
					v.MIGStrategy = "mixed"
				}
			}
			// UseOpenKernelModule (camelCase in recipe)
			if val, ok := st.Data["useOpenKernelModule"]; ok {
				if b, ok := val.Any().(bool); ok {
					v.UseOpenKernelModule = common.BoolToString(b)
				}
			}
			// RDMA support (affects GDS)
			if val, ok := st.Data["rdma"]; ok {
				if b, ok := val.Any().(bool); ok {
					v.EnableGDS = common.BoolToString(b)
				}
			}
		}
	}
}

// extractGPUSettings extracts GPU-related settings from measurements.
func (v *HelmValues) extractGPUSettings(m *measurement.Measurement) {
	for _, st := range m.Subtypes {
		// Recipe uses 'smi' subtype for nvidia-smi output
		if st.Name == "smi" {
			if val, ok := st.Data["driver-version"]; ok {
				if s, ok := val.Any().(string); ok {
					// Only set if not already set from K8s measurements
					if v.DriverVersion == "" {
						v.DriverVersion = s
					}
				}
			}
		}
	}
}

// applyConfigOverrides applies configuration overrides to values.
func (v *HelmValues) applyConfigOverrides(config map[string]string) {
	if val, ok := config["driver_version"]; ok && val != "" {
		v.DriverVersion = val
	}
	if val, ok := config["gpu_operator_version"]; ok && val != "" {
		v.GPUOperatorVersion = val
	}
	if val, ok := config["mig_strategy"]; ok && val != "" {
		v.MIGStrategy = val
	}
	if val, ok := config["enable_gds"]; ok {
		v.EnableGDS = common.BoolToString(common.ParseBoolString(val))
	}
	if val, ok := config["vgpu_license_server"]; ok && val != "" {
		v.VGPULicenseServer = val
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

// applyValueOverrides applies user-specified value overrides from --set flags.
// Uses reflection-based approach to dynamically set any field using dot notation.
// Supports paths like "gds.enabled", "gdrcopy.enabled", "mig.strategy", "driver.version".
func (v *HelmValues) applyValueOverrides(overrides map[string]string) {
	if len(overrides) == 0 {
		return
	}

	// Use reflection-based override utility for dynamic field setting
	if err := common.ApplyValueOverrides(v, overrides); err != nil {
		// Log error but continue - some overrides may have succeeded
		fmt.Printf("Warning: failed to apply some value overrides: %v\n", err)
	}

	// Handle special cases that need custom logic
	for path, value := range overrides {
		// Handle custom labels with nodeSelector prefix
		if len(path) > 13 && path[:13] == "nodeSelector." {
			labelKey := path[13:]
			if v.CustomLabels == nil {
				v.CustomLabels = make(map[string]string)
			}
			v.CustomLabels[labelKey] = value
		}
	}
}

// Validate validates the Helm values.
func (v *HelmValues) Validate() error {
	if v.Namespace == "" {
		return fmt.Errorf("namespace cannot be empty")
	}
	if v.DriverRegistry == "" {
		return fmt.Errorf("driver registry cannot be empty")
	}
	if v.MIGStrategy != "" && v.MIGStrategy != "single" && v.MIGStrategy != "mixed" {
		return fmt.Errorf("invalid MIG strategy: %s (must be single or mixed)", v.MIGStrategy)
	}
	return nil
}
