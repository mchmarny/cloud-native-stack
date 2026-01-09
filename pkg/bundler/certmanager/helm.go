package certmanager

import (
	"fmt"
	"time"

	common "github.com/NVIDIA/cloud-native-stack/pkg/bundler/internal"
	"github.com/NVIDIA/cloud-native-stack/pkg/measurement"
	"github.com/NVIDIA/cloud-native-stack/pkg/recipe"
)

// HelmValues represents the data structure for cert-manager Helm values.
type HelmValues struct {
	Timestamp          string
	Version            string
	RecipeVersion      string
	Namespace          string
	CertManagerVersion common.ValueWithContext
	InstallCRDs        common.ValueWithContext
	EnablePrometheus   common.ValueWithContext

	// Image repositories
	ControllerImage common.ValueWithContext
	WebhookImage    common.ValueWithContext
	CAInjectorImage common.ValueWithContext

	// Controller resources
	ControllerCPURequest    common.ValueWithContext
	ControllerCPULimit      common.ValueWithContext
	ControllerMemoryRequest common.ValueWithContext
	ControllerMemoryLimit   common.ValueWithContext

	// Webhook resources
	WebhookCPURequest    common.ValueWithContext
	WebhookCPULimit      common.ValueWithContext
	WebhookMemoryRequest common.ValueWithContext
	WebhookMemoryLimit   common.ValueWithContext

	// CAInjector resources
	CAInjectorCPURequest    common.ValueWithContext
	CAInjectorCPULimit      common.ValueWithContext
	CAInjectorMemoryRequest common.ValueWithContext
	CAInjectorMemoryLimit   common.ValueWithContext

	// Scheduling
	TolerationKey     common.ValueWithContext
	TolerationValue   common.ValueWithContext
	NodeSelectorKey   common.ValueWithContext
	NodeSelectorValue common.ValueWithContext
	ImagePullSecret   common.ValueWithContext

	CustomLabels map[string]string
}

// GenerateHelmValues generates Helm values from a recipe.
func GenerateHelmValues(recipe *recipe.Recipe, config map[string]string, overrides map[string]string) *HelmValues {
	values := &HelmValues{
		Timestamp:        time.Now().UTC().Format(time.RFC3339),
		Version:          common.GetBundlerVersion(config),
		RecipeVersion:    common.GetRecipeBundlerVersion(recipe.Metadata),
		Namespace:        common.GetConfigValue(config, "namespace", Name),
		InstallCRDs:      common.ValueWithContext{Value: "true"},
		EnablePrometheus: common.ValueWithContext{Value: "true"},
		CustomLabels:     common.ExtractCustomLabels(config),

		// Default image repositories (with version tags)
		ControllerImage: common.ValueWithContext{Value: "nvcr.io/0491946863192633/cert-manager-controller:v1.19.1"},
		WebhookImage:    common.ValueWithContext{Value: "nvcr.io/0491946863192633/cert-manager-webhook:v1.19.1"},
		CAInjectorImage: common.ValueWithContext{Value: "nvcr.io/0491946863192633/cert-manager-cainjector:v1.19.1"},

		// Default controller resources
		ControllerCPURequest:    common.ValueWithContext{Value: "50m"},
		ControllerCPULimit:      common.ValueWithContext{Value: "50m"},
		ControllerMemoryRequest: common.ValueWithContext{Value: "90Mi"},
		ControllerMemoryLimit:   common.ValueWithContext{Value: "90Mi"},

		// Default webhook resources
		WebhookCPURequest:    common.ValueWithContext{Value: "50m"},
		WebhookCPULimit:      common.ValueWithContext{Value: "50m"},
		WebhookMemoryRequest: common.ValueWithContext{Value: "40Mi"},
		WebhookMemoryLimit:   common.ValueWithContext{Value: "40Mi"},

		// Default CA injector resources
		CAInjectorCPURequest:    common.ValueWithContext{Value: "50m"},
		CAInjectorCPULimit:      common.ValueWithContext{Value: "50m"},
		CAInjectorMemoryRequest: common.ValueWithContext{Value: "320Mi"},
		CAInjectorMemoryLimit:   common.ValueWithContext{Value: "320Mi"},

		// Default scheduling
		TolerationKey:     common.ValueWithContext{Value: "dedicated"},
		TolerationValue:   common.ValueWithContext{Value: "system-workload"},
		NodeSelectorKey:   common.ValueWithContext{Value: "nodeGroup"},
		NodeSelectorValue: common.ValueWithContext{Value: "system-cpu"},
		ImagePullSecret:   common.ValueWithContext{Value: "nvidia-ngcuser-pull-secret"},
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

// extractK8sSettings extracts Kubernetes-related settings from measurements.
func (v *HelmValues) extractK8sSettings(m *measurement.Measurement) {
	// First, extract registry URI for cert-manager
	certManagerRegistry := v.extractCertManagerRegistry(m)

	for _, st := range m.Subtypes {
		// Extract context for this subtype
		subtypeContext := common.GetSubtypeContext(st.Context)

		// Extract version information from 'image' subtype
		if st.Name == "image" {
			if val, ok := st.Data["cert-manager"]; ok {
				if s, ok := val.Any().(string); ok {
					ctx := common.GetFieldContext(st.Context, "cert-manager", subtypeContext)
					v.CertManagerVersion = common.ValueWithContext{Value: s, Context: ctx}
				}
			}
			if val, ok := st.Data["cert-manager-controller"]; ok {
				if s, ok := val.Any().(string); ok {
					ctx := common.GetFieldContext(st.Context, "cert-manager-controller", subtypeContext)
					// Construct full image URI: registry/component:version
					v.ControllerImage = common.ValueWithContext{
						Value:   fmt.Sprintf("%s/cert-manager-controller:%s", certManagerRegistry, s),
						Context: ctx,
					}
				}
			}
			if val, ok := st.Data["cert-manager-webhook"]; ok {
				if s, ok := val.Any().(string); ok {
					ctx := common.GetFieldContext(st.Context, "cert-manager-webhook", subtypeContext)
					// Construct full image URI: registry/component:version
					v.WebhookImage = common.ValueWithContext{
						Value:   fmt.Sprintf("%s/cert-manager-webhook:%s", certManagerRegistry, s),
						Context: ctx,
					}
				}
			}
			if val, ok := st.Data["cert-manager-cainjector"]; ok {
				if s, ok := val.Any().(string); ok {
					ctx := common.GetFieldContext(st.Context, "cert-manager-cainjector", subtypeContext)
					// Construct full image URI: registry/component:version
					v.CAInjectorImage = common.ValueWithContext{
						Value:   fmt.Sprintf("%s/cert-manager-cainjector:%s", certManagerRegistry, s),
						Context: ctx,
					}
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
	subtypeContext := common.GetSubtypeContext(st.Context)

	// CRDs and monitoring
	if val, ok := st.Data["install_crds"]; ok {
		if b, ok := val.Any().(bool); ok {
			ctx := common.GetFieldContext(st.Context, "install_crds", subtypeContext)
			v.InstallCRDs = common.ValueWithContext{Value: fmt.Sprintf("%t", b), Context: ctx}
		}
	}
	if val, ok := st.Data["enable_prometheus"]; ok {
		if b, ok := val.Any().(bool); ok {
			ctx := common.GetFieldContext(st.Context, "enable_prometheus", subtypeContext)
			v.EnablePrometheus = common.ValueWithContext{Value: fmt.Sprintf("%t", b), Context: ctx}
		}
	}

	v.extractControllerResources(st, subtypeContext)
	v.extractWebhookResources(st, subtypeContext)
	v.extractCAInjectorResources(st, subtypeContext)
	v.extractScheduling(st, subtypeContext)
}

// extractControllerResources extracts controller resource settings.
//
//nolint:dupl // Intentional duplication - extracting same resource pattern for different components
func (v *HelmValues) extractControllerResources(st *measurement.Subtype, subtypeContext string) {
	if val, ok := st.Data["controller_cpu_request"]; ok {
		if s, ok := val.Any().(string); ok {
			ctx := common.GetFieldContext(st.Context, "controller_cpu_request", subtypeContext)
			v.ControllerCPURequest = common.ValueWithContext{Value: s, Context: ctx}
		}
	}
	if val, ok := st.Data["controller_cpu_limit"]; ok {
		if s, ok := val.Any().(string); ok {
			ctx := common.GetFieldContext(st.Context, "controller_cpu_limit", subtypeContext)
			v.ControllerCPULimit = common.ValueWithContext{Value: s, Context: ctx}
		}
	}
	if val, ok := st.Data["controller_memory_request"]; ok {
		if s, ok := val.Any().(string); ok {
			ctx := common.GetFieldContext(st.Context, "controller_memory_request", subtypeContext)
			v.ControllerMemoryRequest = common.ValueWithContext{Value: s, Context: ctx}
		}
	}
	if val, ok := st.Data["controller_memory_limit"]; ok {
		if s, ok := val.Any().(string); ok {
			ctx := common.GetFieldContext(st.Context, "controller_memory_limit", subtypeContext)
			v.ControllerMemoryLimit = common.ValueWithContext{Value: s, Context: ctx}
		}
	}
}

// extractWebhookResources extracts webhook resource settings.
//
//nolint:dupl // Intentional duplication - extracting same resource pattern for different components
func (v *HelmValues) extractWebhookResources(st *measurement.Subtype, subtypeContext string) {
	if val, ok := st.Data["webhook_cpu_request"]; ok {
		if s, ok := val.Any().(string); ok {
			ctx := common.GetFieldContext(st.Context, "webhook_cpu_request", subtypeContext)
			v.WebhookCPURequest = common.ValueWithContext{Value: s, Context: ctx}
		}
	}
	if val, ok := st.Data["webhook_cpu_limit"]; ok {
		if s, ok := val.Any().(string); ok {
			ctx := common.GetFieldContext(st.Context, "webhook_cpu_limit", subtypeContext)
			v.WebhookCPULimit = common.ValueWithContext{Value: s, Context: ctx}
		}
	}
	if val, ok := st.Data["webhook_memory_request"]; ok {
		if s, ok := val.Any().(string); ok {
			ctx := common.GetFieldContext(st.Context, "webhook_memory_request", subtypeContext)
			v.WebhookMemoryRequest = common.ValueWithContext{Value: s, Context: ctx}
		}
	}
	if val, ok := st.Data["webhook_memory_limit"]; ok {
		if s, ok := val.Any().(string); ok {
			ctx := common.GetFieldContext(st.Context, "webhook_memory_limit", subtypeContext)
			v.WebhookMemoryLimit = common.ValueWithContext{Value: s, Context: ctx}
		}
	}
}

// extractCAInjectorResources extracts CA injector resource settings.
//
//nolint:dupl // Intentional duplication - extracting same resource pattern for different components
func (v *HelmValues) extractCAInjectorResources(st *measurement.Subtype, subtypeContext string) {
	if val, ok := st.Data["cainjector_cpu_request"]; ok {
		if s, ok := val.Any().(string); ok {
			ctx := common.GetFieldContext(st.Context, "cainjector_cpu_request", subtypeContext)
			v.CAInjectorCPURequest = common.ValueWithContext{Value: s, Context: ctx}
		}
	}
	if val, ok := st.Data["cainjector_cpu_limit"]; ok {
		if s, ok := val.Any().(string); ok {
			ctx := common.GetFieldContext(st.Context, "cainjector_cpu_limit", subtypeContext)
			v.CAInjectorCPULimit = common.ValueWithContext{Value: s, Context: ctx}
		}
	}
	if val, ok := st.Data["cainjector_memory_request"]; ok {
		if s, ok := val.Any().(string); ok {
			ctx := common.GetFieldContext(st.Context, "cainjector_memory_request", subtypeContext)
			v.CAInjectorMemoryRequest = common.ValueWithContext{Value: s, Context: ctx}
		}
	}
	if val, ok := st.Data["cainjector_memory_limit"]; ok {
		if s, ok := val.Any().(string); ok {
			ctx := common.GetFieldContext(st.Context, "cainjector_memory_limit", subtypeContext)
			v.CAInjectorMemoryLimit = common.ValueWithContext{Value: s, Context: ctx}
		}
	}
}

// extractScheduling extracts scheduling configuration (tolerations and node selectors).
func (v *HelmValues) extractScheduling(st *measurement.Subtype, subtypeContext string) {
	if val, ok := st.Data["toleration_key"]; ok {
		if s, ok := val.Any().(string); ok {
			ctx := common.GetFieldContext(st.Context, "toleration_key", subtypeContext)
			v.TolerationKey = common.ValueWithContext{Value: s, Context: ctx}
		}
	}
	if val, ok := st.Data["toleration_value"]; ok {
		if s, ok := val.Any().(string); ok {
			ctx := common.GetFieldContext(st.Context, "toleration_value", subtypeContext)
			v.TolerationValue = common.ValueWithContext{Value: s, Context: ctx}
		}
	}
	if val, ok := st.Data["node_selector_key"]; ok {
		if s, ok := val.Any().(string); ok {
			ctx := common.GetFieldContext(st.Context, "node_selector_key", subtypeContext)
			v.NodeSelectorKey = common.ValueWithContext{Value: s, Context: ctx}
		}
	}
	if val, ok := st.Data["node_selector_value"]; ok {
		if s, ok := val.Any().(string); ok {
			ctx := common.GetFieldContext(st.Context, "node_selector_value", subtypeContext)
			v.NodeSelectorValue = common.ValueWithContext{Value: s, Context: ctx}
		}
	}
	if val, ok := st.Data["image_pull_secret"]; ok {
		if s, ok := val.Any().(string); ok {
			ctx := common.GetFieldContext(st.Context, "image_pull_secret", subtypeContext)
			v.ImagePullSecret = common.ValueWithContext{Value: s, Context: ctx}
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
		v.CertManagerVersion = common.ValueWithContext{Value: version}
	}
}

// applyValueOverrides applies value overrides from --set flags.
func (v *HelmValues) applyValueOverrides(overrides map[string]string) {
	if overrides == nil {
		return
	}

	fieldMap := map[string]*common.ValueWithContext{
		"version":                             &v.CertManagerVersion,
		"installCRDs":                         &v.InstallCRDs,
		"prometheus.enabled":                  &v.EnablePrometheus,
		"controller.image":                    &v.ControllerImage,
		"webhook.image":                       &v.WebhookImage,
		"cainjector.image":                    &v.CAInjectorImage,
		"controller.resources.cpu.request":    &v.ControllerCPURequest,
		"controller.resources.cpu.limit":      &v.ControllerCPULimit,
		"controller.resources.memory.request": &v.ControllerMemoryRequest,
		"controller.resources.memory.limit":   &v.ControllerMemoryLimit,
		"webhook.resources.cpu.request":       &v.WebhookCPURequest,
		"webhook.resources.cpu.limit":         &v.WebhookCPULimit,
		"webhook.resources.memory.request":    &v.WebhookMemoryRequest,
		"webhook.resources.memory.limit":      &v.WebhookMemoryLimit,
		"cainjector.resources.cpu.request":    &v.CAInjectorCPURequest,
		"cainjector.resources.cpu.limit":      &v.CAInjectorCPULimit,
		"cainjector.resources.memory.request": &v.CAInjectorMemoryRequest,
		"cainjector.resources.memory.limit":   &v.CAInjectorMemoryLimit,
		"tolerations.key":                     &v.TolerationKey,
		"tolerations.value":                   &v.TolerationValue,
		"nodeSelector.key":                    &v.NodeSelectorKey,
		"nodeSelector.value":                  &v.NodeSelectorValue,
		"imagePullSecret":                     &v.ImagePullSecret,
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
