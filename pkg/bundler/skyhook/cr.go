package skyhook

import (
	"strings"
	"time"

	common "github.com/NVIDIA/cloud-native-stack/pkg/bundler/internal"
	"github.com/NVIDIA/cloud-native-stack/pkg/measurement"
	"github.com/NVIDIA/cloud-native-stack/pkg/recipe"
)

// SkyhookCRData represents the data structure for Skyhook custom resource.
type SkyhookCRData struct {
	Timestamp                 string
	Version                   string
	RecipeVersion             string
	Name                      string
	RuntimeRequired           common.ValueWithContext
	InterruptionBudgetPercent common.ValueWithContext
	NodeSelectorKey           common.ValueWithContext
	NodeSelectorValues        []string // Changed to slice for easier template iteration
	TolerationKey             common.ValueWithContext
	TuningVersion             common.ValueWithContext
	TuningImage               common.ValueWithContext
	TuningInterruptType       common.ValueWithContext
	OperatorRegistry          common.ValueWithContext
	GrubConfig                string
	SysctlConfig              string
	ContainerdServiceConfig   string
}

// GenerateSkyhookCRData generates Skyhook CR data from a recipe.
func GenerateSkyhookCRData(recipe *recipe.Recipe, config map[string]string) *SkyhookCRData {
	// Split node selector values from comma-separated string
	nodeSelectorValuesStr := common.GetConfigValue(config, "node_selector_values", "H100,GB200")
	nodeSelectorValues := strings.Split(nodeSelectorValuesStr, ",")
	// Trim whitespace from each value
	for i, val := range nodeSelectorValues {
		nodeSelectorValues[i] = strings.TrimSpace(val)
	}

	data := &SkyhookCRData{
		Timestamp:                 time.Now().UTC().Format(time.RFC3339),
		Version:                   common.GetBundlerVersion(config),
		RecipeVersion:             common.GetRecipeBundlerVersion(recipe.Metadata),
		Name:                      "skyhook-system",
		RuntimeRequired:           common.ValueWithContext{Value: common.GetConfigValue(config, "runtime_required", "true")},
		InterruptionBudgetPercent: common.ValueWithContext{Value: common.GetConfigValue(config, "interruption_budget_percent", "100")},
		NodeSelectorKey:           common.ValueWithContext{Value: common.GetConfigValue(config, "node_selector", "nvidia.com/gpu.product")},
		NodeSelectorValues:        nodeSelectorValues,
		TolerationKey:             common.ValueWithContext{Value: common.GetConfigValue(config, "toleration_key", "nvidia.com/gpu")},
		TuningInterruptType:       common.ValueWithContext{Value: common.GetConfigValue(config, "tuning_interrupt_type", "reboot")},
		OperatorRegistry:          common.ValueWithContext{Value: common.GetConfigValue(config, "operator_registry", "nvcr.io/nvidia")},
	}

	// Build tuning version and image
	tuningVersion := common.GetConfigValue(config, "tuning_version", "v1.0.0")
	data.TuningVersion = common.ValueWithContext{Value: tuningVersion}
	operatorRegistry := common.GetConfigValue(config, "operator_registry", "nvcr.io/nvidia")
	data.TuningImage = common.ValueWithContext{
		Value: operatorRegistry + "/nodeos-updater-tuning:" + tuningVersion,
	}

	// Extract K8s settings
	for _, m := range recipe.Measurements {
		if m.Type == measurement.TypeK8s {
			data.extractK8sSettings(m)
		}
	}

	// Build configuration strings (pre-indented for template)
	data.GrubConfig = indentLines(buildGrubConfig(recipe), 10)
	data.SysctlConfig = indentLines(buildSysctlConfig(recipe), 10)
	data.ContainerdServiceConfig = indentLines(buildContainerdServiceConfig(recipe), 10)

	return data
}

// indentLines adds indentation to each line of text.
func indentLines(text string, spaces int) string {
	if text == "" {
		return ""
	}
	indent := strings.Repeat(" ", spaces)
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		if line != "" { // Don't indent empty lines
			lines[i] = indent + line
		}
	}
	return strings.Join(lines, "\n")
}

// extractK8sSettings extracts Kubernetes-specific settings.
func (d *SkyhookCRData) extractK8sSettings(m *measurement.Measurement) {
	for _, st := range m.Subtypes {
		if st.Name == skyhookConfigSubtype {
			// Override defaults with recipe values
			if val, ok := st.Data["runtime_required"]; ok {
				if s, ok := val.Any().(string); ok {
					d.RuntimeRequired = common.ValueWithContext{Value: s}
				}
			}
			if val, ok := st.Data["interruption_budget_percent"]; ok {
				if s, ok := val.Any().(string); ok {
					d.InterruptionBudgetPercent = common.ValueWithContext{Value: s}
				}
			}
			if val, ok := st.Data["tuning_interrupt_type"]; ok {
				if s, ok := val.Any().(string); ok {
					d.TuningInterruptType = common.ValueWithContext{Value: s}
				}
			}
		}
	}
}

// buildGrubConfig builds GRUB configuration from OS measurements.
func buildGrubConfig(recipe *recipe.Recipe) string {
	var lines []string

	for _, m := range recipe.Measurements {
		if m.Type != measurement.TypeOS {
			continue
		}

		for _, st := range m.Subtypes {
			if st.Name != "grub" {
				continue
			}

			// Build GRUB parameters
			for key, val := range st.Data {
				if s, ok := val.Any().(string); ok {
					if s == "" {
						// Boolean flag (like nokaslr)
						lines = append(lines, key)
					} else {
						// Key=value parameter
						lines = append(lines, key+"="+s)
					}
				}
			}
		}
	}

	return strings.Join(lines, "\n")
}

// buildSysctlConfig builds sysctl configuration from OS measurements.
func buildSysctlConfig(recipe *recipe.Recipe) string {
	var lines []string

	for _, m := range recipe.Measurements {
		if m.Type != measurement.TypeOS {
			continue
		}

		for _, st := range m.Subtypes {
			if st.Name != "sysctl" {
				continue
			}

			// Convert path format to dot notation
			for path, val := range st.Data {
				if s, ok := val.Any().(string); ok {
					// Convert /proc/sys/kernel/threads-max → kernel.threads-max
					dotKey := convertPathToDotNotation(path)
					lines = append(lines, dotKey+"="+s)
				}
			}
		}
	}

	return strings.Join(lines, "\n")
}

// convertPathToDotNotation converts /proc/sys/kernel/threads-max to kernel.threads-max.
func convertPathToDotNotation(path string) string {
	// Remove /proc/sys/ prefix if present
	path = strings.TrimPrefix(path, "/proc/sys/")
	// Replace / with .
	return strings.ReplaceAll(path, "/", ".")
}

// buildContainerdServiceConfig builds containerd service configuration from SystemD measurements.
func buildContainerdServiceConfig(recipe *recipe.Recipe) string {
	var lines []string
	lines = append(lines, "[Service]")

	for _, m := range recipe.Measurements {
		if m.Type != measurement.TypeSystemD {
			continue
		}

		for _, st := range m.Subtypes {
			if !strings.Contains(st.Name, "containerd") {
				continue
			}

			// Build service configuration
			for key, val := range st.Data {
				if s, ok := val.Any().(string); ok {
					lines = append(lines, key+"="+s)
				}
			}
		}
	}

	return strings.Join(lines, "\n")
}
