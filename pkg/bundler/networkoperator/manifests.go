package networkoperator

import (
	"time"

	common "github.com/NVIDIA/cloud-native-stack/pkg/bundler/internal"
	"github.com/NVIDIA/cloud-native-stack/pkg/measurement"
	"github.com/NVIDIA/cloud-native-stack/pkg/recipe"
)

// ManifestData represents data for generating Kubernetes manifests.
type ManifestData struct {
	Timestamp              string
	DriverRegistry         string
	Namespace              string
	EnableRDMA             bool
	EnableSRIOV            bool
	EnableHostDevice       bool
	EnableIPAM             bool
	DeployOFED             bool
	OFEDVersion            string
	NicType                string
	ContainerRuntimeSocket string
	CustomLabels           map[string]string
	CustomAnnotations      map[string]string
	Version                string
	RecipeVersion          string
}

// GenerateManifestData creates manifest data from a recipe and config.
func GenerateManifestData(recipe *recipe.Recipe, config map[string]string) *ManifestData {
	data := &ManifestData{
		Timestamp:              time.Now().UTC().Format(time.RFC3339),
		DriverRegistry:         common.GetConfigValue(config, "driver_registry", "nvcr.io/nvidia"),
		Namespace:              common.GetConfigValue(config, "namespace", "nvidia-network-operator"),
		EnableRDMA:             false,
		EnableSRIOV:            false,
		EnableHostDevice:       true,
		EnableIPAM:             true,
		DeployOFED:             false,
		NicType:                "ConnectX",
		ContainerRuntimeSocket: "/var/run/containerd/containerd.sock",
		CustomLabels:           common.ExtractCustomLabels(config),
		CustomAnnotations:      common.ExtractCustomAnnotations(config),
		Version:                common.GetBundlerVersion(config),
		RecipeVersion:          common.GetRecipeBundlerVersion(recipe.Metadata),
	}

	// Extract values from recipe (similar to HelmValues)
	// Note: Using nil for overrides here since manifest generation doesn't need custom overrides
	// (manifests use direct recipe values, not user-customizable helm values)
	helmValues := GenerateHelmValues(recipe, config, nil)

	// Convert helm values to manifest data - extract Value from ConfigValue
	if rdma, ok := helmValues.EnableRDMA.Value.(bool); ok {
		data.EnableRDMA = rdma
	}
	if sriov, ok := helmValues.EnableSRIOV.Value.(bool); ok {
		data.EnableSRIOV = sriov
	}
	if hd, ok := helmValues.EnableHostDevice.Value.(bool); ok {
		data.EnableHostDevice = hd
	}
	if ipam, ok := helmValues.EnableIPAM.Value.(bool); ok {
		data.EnableIPAM = ipam
	}
	if ofed, ok := helmValues.DeployOFED.Value.(bool); ok {
		data.DeployOFED = ofed
	}
	if ofedVer, ok := helmValues.OFEDVersion.Value.(string); ok {
		data.OFEDVersion = ofedVer
	}
	if nt, ok := helmValues.NicType.Value.(string); ok {
		data.NicType = nt
	}
	if crs, ok := helmValues.ContainerRuntimeSocket.Value.(string); ok {
		data.ContainerRuntimeSocket = crs
	}
	data.CustomLabels = helmValues.CustomLabels

	// Extract additional settings from K8s config subtype
	for _, m := range recipe.Measurements {
		if m.Type == measurement.TypeK8s {
			for _, st := range m.Subtypes {
				if st.Name == configSubtype {
					// Additional manifest-specific settings can be extracted here
					// Currently all settings are extracted via helm values
					_ = st // Avoid unused variable
				}
			}
		}
	}

	// Custom annotations
	data.CustomAnnotations = common.ExtractCustomAnnotations(config)

	return data
}

// ToMap converts ManifestData to a map for template rendering.
