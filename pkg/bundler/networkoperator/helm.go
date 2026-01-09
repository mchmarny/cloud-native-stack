package networkoperator

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

// HelmValues represents the data structure for Network Operator Helm values.
type HelmValues struct {
	Timestamp              string
	DriverRegistry         common.ValueWithContext
	NetworkOperatorVersion common.ValueWithContext
	OFEDVersion            common.ValueWithContext
	EnableRDMA             common.ValueWithContext
	EnableSRIOV            common.ValueWithContext
	EnableHostDevice       common.ValueWithContext
	EnableIPAM             common.ValueWithContext
	EnableMultus           common.ValueWithContext
	EnableWhereabouts      common.ValueWithContext
	DeployOFED             common.ValueWithContext
	NicType                common.ValueWithContext
	ContainerRuntimeSocket common.ValueWithContext
	CustomLabels           map[string]string
	Namespace              string
	Version                string
	RecipeVersion          string
}

// GenerateHelmValues generates Helm values from a recipe.
func GenerateHelmValues(recipe *recipe.Recipe, config map[string]string, overrides map[string]string) *HelmValues {
	values := &HelmValues{
		Timestamp:              time.Now().UTC().Format(time.RFC3339),
		DriverRegistry:         common.ValueWithContext{Value: common.GetConfigValue(config, "driver_registry", "nvcr.io/nvidia")},
		EnableRDMA:             common.ValueWithContext{Value: false},
		EnableSRIOV:            common.ValueWithContext{Value: false},
		EnableHostDevice:       common.ValueWithContext{Value: true},
		EnableIPAM:             common.ValueWithContext{Value: true},
		EnableMultus:           common.ValueWithContext{Value: true},
		EnableWhereabouts:      common.ValueWithContext{Value: true},
		DeployOFED:             common.ValueWithContext{Value: false},
		NicType:                common.ValueWithContext{Value: "ConnectX"},
		ContainerRuntimeSocket: common.ValueWithContext{Value: "/var/run/containerd/containerd.sock"},
		CustomLabels:           common.ExtractCustomLabels(config),
		Namespace:              common.GetConfigValue(config, "namespace", Name),
		Version:                common.GetBundlerVersion(config),
		RecipeVersion:          common.GetRecipeBundlerVersion(recipe.Metadata),
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

// extractK8sSettings extracts Kubernetes-related settings from measurements.
func (v *HelmValues) extractK8sSettings(m *measurement.Measurement) {
	for _, st := range m.Subtypes {
		// Extract context for this subtype
		subtypeContext := common.GetSubtypeContext(st.Context)

		// Extract version information from 'image' subtype
		if st.Name == "image" {
			if val, ok := st.Data["network-operator"]; ok {
				if s, ok := val.Any().(string); ok {
					ctx := common.GetFieldContext(st.Context, "network-operator", subtypeContext)
					v.NetworkOperatorVersion = common.ValueWithContext{Value: s, Context: ctx}
				}
			}
			if val, ok := st.Data["ofed-driver"]; ok {
				if s, ok := val.Any().(string); ok {
					ctx := common.GetFieldContext(st.Context, "ofed-driver", subtypeContext)
					v.OFEDVersion = common.ValueWithContext{Value: s, Context: ctx}
				}
			}
		}

		// Extract configuration flags from 'config' subtype
		if st.Name == "config" {
			// RDMA configuration
			if val, ok := st.Data["rdma"]; ok {
				if b, ok := val.Any().(bool); ok {
					ctx := common.GetFieldContext(st.Context, "rdma", subtypeContext)
					v.EnableRDMA = common.ValueWithContext{Value: b, Context: ctx}
				}
			}
			// SR-IOV configuration
			if val, ok := st.Data["sr-iov"]; ok {
				if b, ok := val.Any().(bool); ok {
					ctx := common.GetFieldContext(st.Context, "sr-iov", subtypeContext)
					v.EnableSRIOV = common.ValueWithContext{Value: b, Context: ctx}
				}
			}
			// OFED deployment
			if val, ok := st.Data["deploy-ofed"]; ok {
				if b, ok := val.Any().(bool); ok {
					ctx := common.GetFieldContext(st.Context, "deploy-ofed", subtypeContext)
					v.DeployOFED = common.ValueWithContext{Value: b, Context: ctx}
				}
			}
			// Host device plugin
			if val, ok := st.Data["host-device"]; ok {
				if b, ok := val.Any().(bool); ok {
					ctx := common.GetFieldContext(st.Context, "host-device", subtypeContext)
					v.EnableHostDevice = common.ValueWithContext{Value: b, Context: ctx}
				}
			}
			// IPAM plugin
			if val, ok := st.Data["ipam"]; ok {
				if b, ok := val.Any().(bool); ok {
					ctx := common.GetFieldContext(st.Context, "ipam", subtypeContext)
					v.EnableIPAM = common.ValueWithContext{Value: b, Context: ctx}
				}
			}
			// Multus CNI
			if val, ok := st.Data["multus"]; ok {
				if b, ok := val.Any().(bool); ok {
					ctx := common.GetFieldContext(st.Context, "multus", subtypeContext)
					v.EnableMultus = common.ValueWithContext{Value: b, Context: ctx}
				}
			}
			// Whereabouts IPAM
			if val, ok := st.Data["whereabouts"]; ok {
				if b, ok := val.Any().(bool); ok {
					ctx := common.GetFieldContext(st.Context, "whereabouts", subtypeContext)
					v.EnableWhereabouts = common.ValueWithContext{Value: b, Context: ctx}
				}
			}
			// NIC type
			if val, ok := st.Data["nic-type"]; ok {
				if s, ok := val.Any().(string); ok {
					ctx := common.GetFieldContext(st.Context, "nic-type", subtypeContext)
					v.NicType = common.ValueWithContext{Value: s, Context: ctx}
				}
			}
		}

		// Extract container runtime from 'server' subtype
		if st.Name == "server" {
			if val, ok := st.Data["container-runtime"]; ok {
				if s, ok := val.Any().(string); ok {
					ctx := common.GetFieldContext(st.Context, "container-runtime", subtypeContext)
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
					v.ContainerRuntimeSocket = common.ValueWithContext{Value: socket, Context: ctx}
				}
			}
		}
	}
}

// applyConfigOverrides applies configuration overrides to values.
func (v *HelmValues) applyConfigOverrides(config map[string]string) {
	if val, ok := config["network_operator_version"]; ok && val != "" {
		v.NetworkOperatorVersion = common.ValueWithContext{Value: val, Context: "Override from bundler configuration"}
	}
	if val, ok := config["ofed_version"]; ok && val != "" {
		v.OFEDVersion = common.ValueWithContext{Value: val, Context: "Override from bundler configuration"}
	}
	if val, ok := config["enable_rdma"]; ok {
		v.EnableRDMA = common.ValueWithContext{Value: val == strTrue, Context: "Override from bundler configuration"}
	}
	if val, ok := config["enable_sriov"]; ok {
		v.EnableSRIOV = common.ValueWithContext{Value: val == strTrue, Context: "Override from bundler configuration"}
	}
	if val, ok := config["deploy_ofed"]; ok {
		v.DeployOFED = common.ValueWithContext{Value: val == strTrue, Context: "Override from bundler configuration"}
	}
	if val, ok := config["enable_host_device"]; ok {
		v.EnableHostDevice = common.ValueWithContext{Value: val == strTrue, Context: "Override from bundler configuration"}
	}
	if val, ok := config["enable_ipam"]; ok {
		v.EnableIPAM = common.ValueWithContext{Value: val == strTrue, Context: "Override from bundler configuration"}
	}
	if val, ok := config["enable_multus"]; ok {
		v.EnableMultus = common.ValueWithContext{Value: val == strTrue, Context: "Override from bundler configuration"}
	}
	if val, ok := config["enable_whereabouts"]; ok {
		v.EnableWhereabouts = common.ValueWithContext{Value: val == strTrue, Context: "Override from bundler configuration"}
	}
	if val, ok := config["nic_type"]; ok && val != "" {
		v.NicType = common.ValueWithContext{Value: val, Context: "Override from bundler configuration"}
	}
	if val, ok := config["container_runtime_socket"]; ok && val != "" {
		v.ContainerRuntimeSocket = common.ValueWithContext{Value: val, Context: "Override from bundler configuration"}
	}
	if val, ok := config["namespace"]; ok && val != "" {
		v.Namespace = val
	}

	// Custom labels
	v.CustomLabels = common.ExtractCustomLabels(config)
}

// applyValueOverrides applies value overrides from --set flags.
func (v *HelmValues) applyValueOverrides(overrides map[string]string) {
	if overrides == nil {
		return
	}

	fieldMap := map[string]*common.ValueWithContext{
		"driver.registry":         &v.DriverRegistry,
		"operator.version":        &v.NetworkOperatorVersion,
		"ofed.version":            &v.OFEDVersion,
		"rdma.enabled":            &v.EnableRDMA,
		"sriov.enabled":           &v.EnableSRIOV,
		"hostDevice.enabled":      &v.EnableHostDevice,
		"ipam.enabled":            &v.EnableIPAM,
		"multus.enabled":          &v.EnableMultus,
		"whereabouts.enabled":     &v.EnableWhereabouts,
		"ofed.deploy":             &v.DeployOFED,
		"nic.type":                &v.NicType,
		"containerRuntime.socket": &v.ContainerRuntimeSocket,
	}

	// Apply overrides
	for path, value := range overrides {
		if field, exists := fieldMap[path]; exists {
			*field = common.ValueWithContext{
				Value:   value,
				Context: "User override via --set flag",
			}
		}
	}

	// Handle namespace separately (it's a string, not ValueWithContext)
	if ns, exists := overrides["namespace"]; exists {
		v.Namespace = ns
	}
}

// ToMap converts HelmValues to a map for template rendering.
// Validate validates the Helm values.
func (v *HelmValues) Validate() error {
	if v.Namespace == "" {
		return fmt.Errorf("namespace cannot be empty")
	}
	if nt, ok := v.NicType.Value.(string); !ok || nt == "" {
		return fmt.Errorf("NIC type cannot be empty")
	}
	if crs, ok := v.ContainerRuntimeSocket.Value.(string); !ok || crs == "" {
		return fmt.Errorf("container runtime socket cannot be empty")
	}
	return nil
}
