package networkoperator

import (
	"fmt"
	"time"

	common "github.com/NVIDIA/cloud-native-stack/pkg/component/internal"
	"github.com/NVIDIA/cloud-native-stack/pkg/measurement"
	"github.com/NVIDIA/cloud-native-stack/pkg/recipe"
)

// HelmValues represents the data structure for Network Operator Helm values.
type HelmValues struct {
	Timestamp              string
	DriverRegistry         string
	NetworkOperatorVersion string
	OFEDVersion            string
	EnableRDMA             string
	EnableSRIOV            string
	EnableHostDevice       string
	EnableIPAM             string
	EnableMultus           string
	EnableWhereabouts      string
	DeployOFED             string
	NicType                string
	ContainerRuntimeSocket string
	CustomLabels           map[string]string
	Namespace              string
	Version                string
	RecipeVersion          string
}

// GenerateHelmValues generates Helm values from a recipe.
func GenerateHelmValues(recipe *recipe.Recipe, config map[string]string, overrides map[string]string) *HelmValues {
	values := &HelmValues{
		Timestamp:              time.Now().UTC().Format(time.RFC3339),
		DriverRegistry:         common.GetConfigValue(config, "driver_registry", "nvcr.io/nvidia"),
		EnableRDMA:             common.StrFalse,
		EnableSRIOV:            common.StrFalse,
		EnableHostDevice:       common.StrTrue,
		EnableIPAM:             common.StrTrue,
		EnableMultus:           common.StrTrue,
		EnableWhereabouts:      common.StrTrue,
		DeployOFED:             common.StrFalse,
		NicType:                "ConnectX",
		ContainerRuntimeSocket: "/var/run/containerd/containerd.sock",
		CustomLabels:           common.ExtractCustomLabels(config),
		Namespace:              common.GetConfigValue(config, "namespace", Name),
		Version:                common.GetBundlerVersion(config),
		RecipeVersion:          common.GetRecipeBundlerVersion(config),
	}

	// Extract Network Operator configuration from recipe measurements
	for _, m := range recipe.Measurements {
		switch m.Type {
		case measurement.TypeK8s:
			values.extractK8sSettings(m)
		case measurement.TypeSystemD, measurement.TypeOS, measurement.TypeGPU:
			// Not used for Helm values generation
		}
	}

	// Apply config overrides
	values.applyConfigOverrides(config)

	// Apply value overrides from --set flags
	values.applyValueOverrides(overrides)

	return values
}

// GenerateHelmValuesFromMap generates Helm values from config map (for RecipeResult inputs).
func GenerateHelmValuesFromMap(config map[string]string) *HelmValues {
	helmValues := &HelmValues{
		Timestamp:        time.Now().UTC().Format(time.RFC3339),
		EnableRDMA:       common.StrFalse,
		EnableSRIOV:      common.StrFalse,
		EnableHostDevice: common.StrFalse,
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
			if val, ok := st.Data["network-operator"]; ok {
				if s, ok := val.Any().(string); ok {
					v.NetworkOperatorVersion = s
				}
			}
			if val, ok := st.Data["ofed-driver"]; ok {
				if s, ok := val.Any().(string); ok {
					v.OFEDVersion = s
				}
			}
		}

		// Extract configuration flags from 'config' subtype
		if st.Name == "config" {
			// RDMA configuration
			if val, ok := st.Data["rdma"]; ok {
				if b, ok := val.Any().(bool); ok {
					v.EnableRDMA = common.BoolToString(b)
				}
			}
			// SR-IOV configuration
			if val, ok := st.Data["sr-iov"]; ok {
				if b, ok := val.Any().(bool); ok {
					v.EnableSRIOV = common.BoolToString(b)
				}
			}
			// OFED deployment
			if val, ok := st.Data["deploy-ofed"]; ok {
				if b, ok := val.Any().(bool); ok {
					v.DeployOFED = common.BoolToString(b)
				}
			}
			// Host device plugin
			if val, ok := st.Data["host-device"]; ok {
				if b, ok := val.Any().(bool); ok {
					v.EnableHostDevice = common.BoolToString(b)
				}
			}
			// IPAM plugin
			if val, ok := st.Data["ipam"]; ok {
				if b, ok := val.Any().(bool); ok {
					v.EnableIPAM = common.BoolToString(b)
				}
			}
			// Multus CNI
			if val, ok := st.Data["multus"]; ok {
				if b, ok := val.Any().(bool); ok {
					v.EnableMultus = common.BoolToString(b)
				}
			}
			// Whereabouts IPAM
			if val, ok := st.Data["whereabouts"]; ok {
				if b, ok := val.Any().(bool); ok {
					v.EnableWhereabouts = common.BoolToString(b)
				}
			}
			// NIC type
			if val, ok := st.Data["nic-type"]; ok {
				if s, ok := val.Any().(string); ok {
					v.NicType = s
				}
			}
		}

		// Extract container runtime from 'server' subtype
		if st.Name == "server" {
			if val, ok := st.Data["container-runtime"]; ok {
				if s, ok := val.Any().(string); ok {
					var socket string
					switch s {
					case "containerd":
						socket = "/var/run/containerd/containerd.sock"
					case "docker":
						socket = "/var/run/docker.sock"
					case "cri-o":
						socket = "/var/run/crio/crio.sock"
					default:
						socket = "/var/run/containerd/containerd.sock"
					}
					v.ContainerRuntimeSocket = socket
				}
			}
		}
	}
}

// applyConfigOverrides applies configuration overrides to values.
func (v *HelmValues) applyConfigOverrides(config map[string]string) {
	if val, ok := config["network_operator_version"]; ok && val != "" {
		v.NetworkOperatorVersion = val
	}
	if val, ok := config["ofed_version"]; ok && val != "" {
		v.OFEDVersion = val
	}
	if val, ok := config["enable_rdma"]; ok {
		v.EnableRDMA = common.BoolToString(common.ParseBoolString(val))
	}
	if val, ok := config["enable_sriov"]; ok {
		v.EnableSRIOV = common.BoolToString(common.ParseBoolString(val))
	}
	if val, ok := config["deploy_ofed"]; ok {
		v.DeployOFED = common.BoolToString(common.ParseBoolString(val))
	}
	if val, ok := config["enable_host_device"]; ok {
		v.EnableHostDevice = common.BoolToString(common.ParseBoolString(val))
	}
	if val, ok := config["enable_ipam"]; ok {
		v.EnableIPAM = common.BoolToString(common.ParseBoolString(val))
	}
	if val, ok := config["enable_multus"]; ok {
		v.EnableMultus = common.BoolToString(common.ParseBoolString(val))
	}
	if val, ok := config["enable_whereabouts"]; ok {
		v.EnableWhereabouts = common.BoolToString(common.ParseBoolString(val))
	}
	if val, ok := config["nic_type"]; ok && val != "" {
		v.NicType = val
	}
	if val, ok := config["container_runtime_socket"]; ok && val != "" {
		v.ContainerRuntimeSocket = val
	}
	if val, ok := config["namespace"]; ok && val != "" {
		v.Namespace = val
	}

	// Custom labels
	v.CustomLabels = common.ExtractCustomLabels(config)
}

// applyValueOverrides applies value overrides from --set flags.
// Uses reflection-based approach to dynamically set any field using dot notation.
// Supports paths like "rdma.enabled", "operator.version", "ofed.version".
func (v *HelmValues) applyValueOverrides(overrides map[string]string) {
	if len(overrides) == 0 {
		return
	}

	// Use reflection-based override utility for dynamic field setting
	if err := common.ApplyValueOverrides(v, overrides); err != nil {
		// Log error but continue - some overrides may have succeeded
		fmt.Printf("Warning: failed to apply some value overrides: %v\n", err)
	}
}

// Validate validates the Helm values.
func (v *HelmValues) Validate() error {
	if v.Namespace == "" {
		return fmt.Errorf("namespace cannot be empty")
	}
	if v.NicType == "" {
		return fmt.Errorf("NIC type cannot be empty")
	}
	if v.ContainerRuntimeSocket == "" {
		return fmt.Errorf("container runtime socket cannot be empty")
	}
	return nil
}
