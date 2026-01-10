package certmanager

import (
	"fmt"
	"time"

	common "github.com/NVIDIA/cloud-native-stack/pkg/component/internal"
	"github.com/NVIDIA/cloud-native-stack/pkg/measurement"
	"github.com/NVIDIA/cloud-native-stack/pkg/recipe"
)

// HelmValues represents the data structure for cert-manager Helm values.
type HelmValues struct {
	Timestamp          string
	Version            string
	RecipeVersion      string
	Namespace          string
	CertManagerVersion string
	InstallCRDs        string
	EnablePrometheus   string

	// Image repositories
	ControllerImage string
	WebhookImage    string
	CAInjectorImage string

	// Controller resources
	ControllerCPURequest    string
	ControllerCPULimit      string
	ControllerMemoryRequest string
	ControllerMemoryLimit   string

	// Webhook resources
	WebhookCPURequest    string
	WebhookCPULimit      string
	WebhookMemoryRequest string
	WebhookMemoryLimit   string

	// CAInjector resources
	CAInjectorCPURequest    string
	CAInjectorCPULimit      string
	CAInjectorMemoryRequest string
	CAInjectorMemoryLimit   string

	// Scheduling
	TolerationKey     string
	TolerationValue   string
	NodeSelectorKey   string
	NodeSelectorValue string
	ImagePullSecret   string

	CustomLabels map[string]string
}

// GenerateHelmValues generates Helm values from a recipe.
func GenerateHelmValues(recipe *recipe.Recipe, config map[string]string, overrides map[string]string) *HelmValues {
	values := &HelmValues{
		Timestamp:        time.Now().UTC().Format(time.RFC3339),
		Version:          common.GetBundlerVersion(config),
		RecipeVersion:    common.GetRecipeBundlerVersion(config),
		Namespace:        common.GetConfigValue(config, "namespace", Name),
		InstallCRDs:      "true",
		EnablePrometheus: "true",
		CustomLabels:     common.ExtractCustomLabels(config),

		// Default image repositories (with version tags)
		ControllerImage: "nvcr.io/0491946863192633/cert-manager-controller:v1.19.1",
		WebhookImage:    "nvcr.io/0491946863192633/cert-manager-webhook:v1.19.1",
		CAInjectorImage: "nvcr.io/0491946863192633/cert-manager-cainjector:v1.19.1",

		// Default controller resources
		ControllerCPURequest:    "50m",
		ControllerCPULimit:      "50m",
		ControllerMemoryRequest: "90Mi",
		ControllerMemoryLimit:   "90Mi",

		// Default webhook resources
		WebhookCPURequest:    "50m",
		WebhookCPULimit:      "50m",
		WebhookMemoryRequest: "40Mi",
		WebhookMemoryLimit:   "40Mi",

		// Default CA injector resources
		CAInjectorCPURequest:    "50m",
		CAInjectorCPULimit:      "50m",
		CAInjectorMemoryRequest: "320Mi",
		CAInjectorMemoryLimit:   "320Mi",

		// Default scheduling
		TolerationKey:     "dedicated",
		TolerationValue:   "system-workload",
		NodeSelectorKey:   "nodeGroup",
		NodeSelectorValue: "system-cpu",
		ImagePullSecret:   "nvidia-ngcuser-pull-secret",
	}

	// Extract cert-manager configuration from recipe measurements
	for _, m := range recipe.Measurements {
		switch m.Type {
		case measurement.TypeK8s:
			values.extractK8sSettings(m)
		case measurement.TypeGPU, measurement.TypeOS, measurement.TypeSystemD:
			// Not used by cert-manager bundler
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
		Timestamp:          time.Now().UTC().Format(time.RFC3339),
		Version:            common.GetBundlerVersion(config),
		RecipeVersion:      common.GetRecipeBundlerVersion(config),
		Namespace:          common.GetConfigValue(config, "namespace", Name),
		InstallCRDs:        "true",
		EnablePrometheus:   "true",
		CertManagerVersion: common.GetConfigValue(config, "helm_chart_version", "v1.19.1"),
		ControllerImage:    "nvcr.io/0491946863192633/cert-manager-controller:v1.19.1",
		WebhookImage:       "nvcr.io/0491946863192633/cert-manager-webhook:v1.19.1",
		CAInjectorImage:    "nvcr.io/0491946863192633/cert-manager-cainjector:v1.19.1",
	}

	return helmValues
}

// extractK8sSettings extracts Kubernetes-related settings from measurements.
func (v *HelmValues) extractK8sSettings(m *measurement.Measurement) {
	// First, extract registry URI for cert-manager
	certManagerRegistry := v.extractCertManagerRegistry(m)

	for _, st := range m.Subtypes {
		// Extract version information from 'image' subtype
		if st.Name == "image" {
			if val, ok := st.Data["cert-manager"]; ok {
				if s, ok := val.Any().(string); ok {
					v.CertManagerVersion = s
				}
			}
			if val, ok := st.Data["cert-manager-controller"]; ok {
				if s, ok := val.Any().(string); ok {
					v.ControllerImage = fmt.Sprintf("%s/cert-manager-controller:%s", certManagerRegistry, s)
				}
			}
			if val, ok := st.Data["cert-manager-webhook"]; ok {
				if s, ok := val.Any().(string); ok {
					v.WebhookImage = fmt.Sprintf("%s/cert-manager-webhook:%s", certManagerRegistry, s)
				}
			}
			if val, ok := st.Data["cert-manager-cainjector"]; ok {
				if s, ok := val.Any().(string); ok {
					v.CAInjectorImage = fmt.Sprintf("%s/cert-manager-cainjector:%s", certManagerRegistry, s)
				}
			}
		}

		// Extract configuration from 'cert-manager-config' subtype
		if st.Name == "cert-manager-config" {
			v.extractCertManagerConfig(&st)
		}
	}
}

// extractCertManagerRegistry extracts the cert-manager registry URI.
func (v *HelmValues) extractCertManagerRegistry(m *measurement.Measurement) string {
	for _, st := range m.Subtypes {
		if st.Name == "registry" {
			if val, ok := st.Data["cert-manager-uri"]; ok {
				if s, ok := val.Any().(string); ok {
					return s
				}
			}
			break
		}
	}
	// If registry not found, use default
	return "nvcr.io/0491946863192633"
}

// extractCertManagerConfig extracts configuration from cert-manager-config subtype.
func (v *HelmValues) extractCertManagerConfig(st *measurement.Subtype) {
	// CRDs and monitoring
	if val, ok := st.Data["install_crds"]; ok {
		if b, ok := val.Any().(bool); ok {
			v.InstallCRDs = fmt.Sprintf("%t", b)
		}
	}
	if val, ok := st.Data["enable_prometheus"]; ok {
		if b, ok := val.Any().(bool); ok {
			v.EnablePrometheus = fmt.Sprintf("%t", b)
		}
	}

	v.extractControllerResources(st)
	v.extractWebhookResources(st)
	v.extractCAInjectorResources(st)
	v.extractScheduling(st)
}

// extractControllerResources extracts controller resource settings.
func (v *HelmValues) extractControllerResources(st *measurement.Subtype) {
	if val, ok := st.Data["controller_cpu_request"]; ok {
		if s, ok := val.Any().(string); ok {
			v.ControllerCPURequest = s
		}
	}
	if val, ok := st.Data["controller_cpu_limit"]; ok {
		if s, ok := val.Any().(string); ok {
			v.ControllerCPULimit = s
		}
	}
	if val, ok := st.Data["controller_memory_request"]; ok {
		if s, ok := val.Any().(string); ok {
			v.ControllerMemoryRequest = s
		}
	}
	if val, ok := st.Data["controller_memory_limit"]; ok {
		if s, ok := val.Any().(string); ok {
			v.ControllerMemoryLimit = s
		}
	}
}

// extractWebhookResources extracts webhook resource settings.
func (v *HelmValues) extractWebhookResources(st *measurement.Subtype) {
	if val, ok := st.Data["webhook_cpu_request"]; ok {
		if s, ok := val.Any().(string); ok {
			v.WebhookCPURequest = s
		}
	}
	if val, ok := st.Data["webhook_cpu_limit"]; ok {
		if s, ok := val.Any().(string); ok {
			v.WebhookCPULimit = s
		}
	}
	if val, ok := st.Data["webhook_memory_request"]; ok {
		if s, ok := val.Any().(string); ok {
			v.WebhookMemoryRequest = s
		}
	}
	if val, ok := st.Data["webhook_memory_limit"]; ok {
		if s, ok := val.Any().(string); ok {
			v.WebhookMemoryLimit = s
		}
	}
}

// extractCAInjectorResources extracts CA injector resource settings.
func (v *HelmValues) extractCAInjectorResources(st *measurement.Subtype) {
	if val, ok := st.Data["cainjector_cpu_request"]; ok {
		if s, ok := val.Any().(string); ok {
			v.CAInjectorCPURequest = s
		}
	}
	if val, ok := st.Data["cainjector_cpu_limit"]; ok {
		if s, ok := val.Any().(string); ok {
			v.CAInjectorCPULimit = s
		}
	}
	if val, ok := st.Data["cainjector_memory_request"]; ok {
		if s, ok := val.Any().(string); ok {
			v.CAInjectorMemoryRequest = s
		}
	}
	if val, ok := st.Data["cainjector_memory_limit"]; ok {
		if s, ok := val.Any().(string); ok {
			v.CAInjectorMemoryLimit = s
		}
	}
}

// extractScheduling extracts scheduling configuration (tolerations and node selectors).
func (v *HelmValues) extractScheduling(st *measurement.Subtype) {
	if val, ok := st.Data["toleration_key"]; ok {
		if s, ok := val.Any().(string); ok {
			v.TolerationKey = s
		}
	}
	if val, ok := st.Data["toleration_value"]; ok {
		if s, ok := val.Any().(string); ok {
			v.TolerationValue = s
		}
	}
	if val, ok := st.Data["node_selector_key"]; ok {
		if s, ok := val.Any().(string); ok {
			v.NodeSelectorKey = s
		}
	}
	if val, ok := st.Data["node_selector_value"]; ok {
		if s, ok := val.Any().(string); ok {
			v.NodeSelectorValue = s
		}
	}
	if val, ok := st.Data["image_pull_secret"]; ok {
		if s, ok := val.Any().(string); ok {
			v.ImagePullSecret = s
		}
	}
}

// applyConfigOverrides applies configuration overrides from the config map.
func (v *HelmValues) applyConfigOverrides(config map[string]string) {
	// Override namespace if specified
	if ns := common.GetConfigValue(config, "namespace", ""); ns != "" {
		v.Namespace = ns
	}

	// Override cert-manager version if specified
	if version := common.GetConfigValue(config, "cert_manager_version", ""); version != "" {
		v.CertManagerVersion = version
	}
}

// applyValueOverrides applies value overrides from --set flags.
// Uses reflection-based approach to dynamically set any field using dot notation.
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
