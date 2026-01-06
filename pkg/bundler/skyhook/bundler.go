package skyhook

import (
	"context"

	"log/slog"
	"path/filepath"
	"strings"
	"time"

	"github.com/NVIDIA/cloud-native-stack/pkg/bundler/config"
	common "github.com/NVIDIA/cloud-native-stack/pkg/bundler/internal"
	"github.com/NVIDIA/cloud-native-stack/pkg/bundler/result"
	"github.com/NVIDIA/cloud-native-stack/pkg/bundler/types"
	"github.com/NVIDIA/cloud-native-stack/pkg/errors"
	"github.com/NVIDIA/cloud-native-stack/pkg/measurement"
	"github.com/NVIDIA/cloud-native-stack/pkg/recipe"
)

const (
	Name = "skyhook"

	configSubtype        = "config"
	skyhookConfigSubtype = "skyhook-config"
)

// Bundler creates Skyhook Operator application bundles based on recipes.
type Bundler struct {
	*common.BaseBundler
}

// NewBundler creates a new Skyhook bundler instance.
func NewBundler(conf *config.Config) *Bundler {
	return &Bundler{
		BaseBundler: common.NewBaseBundler(conf, types.BundleTypeSkyhook),
	}
}

// Make generates the Skyhook bundle based on the provided recipe.
func (b *Bundler) Make(ctx context.Context, r *recipe.Recipe, dir string) (*result.Result, error) {
	// Check for required measurements
	if err := r.ValidateMeasurementExists(measurement.TypeK8s); err != nil {
		return nil, errors.Wrap(errors.ErrCodeInvalidRequest,
			"K8s measurements are required for Skyhook bundling", err)
	}

	start := time.Now()

	slog.Debug("generating Skyhook bundle",
		"output_dir", dir,
		"namespace", Name,
	)

	// Create bundle directory structure
	dirs, err := b.CreateBundleDir(dir, Name)
	if err != nil {
		return b.Result, errors.Wrap(errors.ErrCodeInternal,
			"failed to create bundle directory", err)
	}

	// Prepare configuration map
	configMap := b.buildConfigMap(r)

	// Generate Helm values
	if err := b.generateHelmValues(ctx, r, dirs.Root, configMap); err != nil {
		return b.Result, err
	}

	// Generate Skyhook CR manifest
	if err := b.generateSkyhookCR(ctx, r, dirs.Manifests, configMap); err != nil {
		return b.Result, err
	}

	// Generate installation scripts
	if b.Config.IncludeScripts() {
		if err := b.generateScripts(ctx, r, dirs.Scripts, configMap); err != nil {
			return b.Result, err
		}
	}

	// Generate README
	if b.Config.IncludeReadme() {
		if err := b.generateReadme(ctx, r, dirs.Root, configMap); err != nil {
			return b.Result, err
		}
	}

	// Generate checksums file
	if b.Config.IncludeChecksums() {
		if err := b.GenerateChecksums(ctx, dirs.Root); err != nil {
			return b.Result, errors.Wrap(errors.ErrCodeInternal,
				"failed to generate checksums", err)
		}
	}

	// Finalize bundle generation
	b.Finalize(start)

	slog.Debug("Skyhook bundle generated",
		"files", len(b.Result.Files),
		"size_bytes", b.Result.Size,
		"duration", b.Result.Duration.Round(time.Millisecond),
	)

	return b.Result, nil
}

// buildConfigMap extracts configuration from recipe and bundler config.
func (b *Bundler) buildConfigMap(r *recipe.Recipe) map[string]string {
	// Start with base config
	configMap := b.BuildBaseConfigMap()
	configMap["namespace"] = Name

	// Extract values from recipe measurements
	for _, m := range r.Measurements {
		switch m.Type {
		case measurement.TypeK8s:
			b.extractK8sConfig(m, configMap)
		case measurement.TypeOS:
			b.extractOSConfig(m, configMap)
		case measurement.TypeSystemD:
			b.extractSystemDConfig(m, configMap)
		case measurement.TypeGPU:
			// GPU measurements not used by Skyhook
		}
	}

	return configMap
}

// extractK8sConfig extracts Kubernetes-related configuration.
func (b *Bundler) extractK8sConfig(m *measurement.Measurement, configMap map[string]string) {
	for _, st := range m.Subtypes {
		if st.Name == "image" {
			// Skyhook Operator version
			if val, ok := st.Data["skyhook-operator"]; ok {
				if s, ok := val.Any().(string); ok {
					configMap["skyhook_operator_version"] = s
				}
			}
			// Skyhook Agent image
			if val, ok := st.Data["skyhook-agent"]; ok {
				if s, ok := val.Any().(string); ok {
					configMap["skyhook_agent_version"] = s
				}
			}
			// Kube RBAC Proxy version
			if val, ok := st.Data["kube-rbac-proxy"]; ok {
				if s, ok := val.Any().(string); ok {
					configMap["kube_rbac_proxy_version"] = s
				}
			}
			// NodeOS Updater Tuning version
			if val, ok := st.Data["nodeos-updater-tuning"]; ok {
				if s, ok := val.Any().(string); ok {
					configMap["tuning_version"] = s
				}
			}
		}

		if st.Name == "registry" {
			// Registry URI
			if val, ok := st.Data["uri"]; ok {
				if s, ok := val.Any().(string); ok {
					configMap["operator_registry"] = s
				}
			}
		}

		if st.Name == skyhookConfigSubtype {
			// Runtime required
			if val, ok := st.Data["runtime_required"]; ok {
				if s, ok := val.Any().(string); ok {
					configMap["runtime_required"] = s
				}
			}
			// Interruption budget percent
			if val, ok := st.Data["interruption_budget_percent"]; ok {
				if s, ok := val.Any().(string); ok {
					configMap["interruption_budget_percent"] = s
				}
			}
			// Tuning interrupt type
			if val, ok := st.Data["tuning_interrupt_type"]; ok {
				if s, ok := val.Any().(string); ok {
					configMap["tuning_interrupt_type"] = s
				}
			}
			// Manager resource limits and requests
			if val, ok := st.Data["manager_cpu_limit"]; ok {
				if s, ok := val.Any().(string); ok {
					configMap["manager_cpu_limit"] = s
				}
			}
			if val, ok := st.Data["manager_memory_limit"]; ok {
				if s, ok := val.Any().(string); ok {
					configMap["manager_memory_limit"] = s
				}
			}
			if val, ok := st.Data["manager_cpu_request"]; ok {
				if s, ok := val.Any().(string); ok {
					configMap["manager_cpu_request"] = s
				}
			}
			if val, ok := st.Data["manager_memory_request"]; ok {
				if s, ok := val.Any().(string); ok {
					configMap["manager_memory_request"] = s
				}
			}
			// Node selector and tolerations
			if val, ok := st.Data["node_selector"]; ok {
				if s, ok := val.Any().(string); ok {
					configMap["node_selector"] = s
				}
			}
			if val, ok := st.Data["node_selector_values"]; ok {
				if s, ok := val.Any().(string); ok {
					configMap["node_selector_values"] = s
				}
			}
			if val, ok := st.Data["toleration_key"]; ok {
				if s, ok := val.Any().(string); ok {
					configMap["toleration_key"] = s
				}
			}
			if val, ok := st.Data["toleration_value"]; ok {
				if s, ok := val.Any().(string); ok {
					configMap["toleration_value"] = s
				}
			}
		}
	}
}

// extractOSConfig extracts OS-related configuration for GRUB and sysctl.
func (b *Bundler) extractOSConfig(m *measurement.Measurement, configMap map[string]string) {
	for _, st := range m.Subtypes {
		if st.Name == "grub" {
			// Extract GRUB settings
			for key, val := range st.Data {
				if s, ok := val.Any().(string); ok {
					configMap["grub_"+key] = s
				}
			}
		}

		if st.Name == "sysctl" {
			// Extract sysctl settings
			for key, val := range st.Data {
				if s, ok := val.Any().(string); ok {
					// Convert path format to key format
					cleanKey := strings.ReplaceAll(key, "/proc/sys/", "")
					cleanKey = strings.ReplaceAll(cleanKey, "/", "_")
					configMap["sysctl_"+cleanKey] = s
				}
			}
		}
	}
}

// extractSystemDConfig extracts SystemD service configuration.
func (b *Bundler) extractSystemDConfig(m *measurement.Measurement, configMap map[string]string) {
	for _, st := range m.Subtypes {
		if strings.Contains(st.Name, "containerd") {
			// Extract containerd service settings
			for key, val := range st.Data {
				if s, ok := val.Any().(string); ok {
					configMap["containerd_"+key] = s
				}
			}
		}
	}
}

// generateHelmValues generates Helm values file.
func (b *Bundler) generateHelmValues(ctx context.Context, r *recipe.Recipe,
	bundleDir string, config map[string]string) error {

	helmValues := GenerateHelmValues(r, config)

	filePath := filepath.Join(bundleDir, "values.yaml")
	return b.GenerateFileFromTemplate(ctx, GetTemplate, "values.yaml",
		filePath, helmValues, 0644)
}

// generateSkyhookCR generates Skyhook custom resource manifest.
func (b *Bundler) generateSkyhookCR(ctx context.Context, r *recipe.Recipe,
	dir string, config map[string]string) error {

	crData := GenerateSkyhookCRData(r, config)
	filePath := filepath.Join(dir, "skyhook.yaml")

	return b.GenerateFileFromTemplate(ctx, GetTemplate, "skyhook.yaml",
		filePath, crData, 0644)
}

// generateScripts generates installation and uninstallation scripts.
func (b *Bundler) generateScripts(ctx context.Context, r *recipe.Recipe,
	dir string, config map[string]string) error {

	scriptData := GenerateScriptData(r, config)

	// Generate install script
	installPath := filepath.Join(dir, "install.sh")
	if err := b.GenerateFileFromTemplate(ctx, GetTemplate, "install.sh",
		installPath, scriptData, 0755); err != nil {
		return err
	}

	// Generate uninstall script
	uninstallPath := filepath.Join(dir, "uninstall.sh")
	return b.GenerateFileFromTemplate(ctx, GetTemplate, "uninstall.sh",
		uninstallPath, scriptData, 0755)
}

// generateReadme generates README documentation.
func (b *Bundler) generateReadme(ctx context.Context, recipe *recipe.Recipe,
	dir string, config map[string]string) error {

	helmValues := GenerateHelmValues(recipe, config)
	crData := GenerateSkyhookCRData(recipe, config)
	scriptData := GenerateScriptData(recipe, config)

	// Combine data structures for README
	data := map[string]interface{}{
		"Helm":     helmValues,
		"CR":       crData,
		"Script":   scriptData,
		"Metadata": recipe.Metadata,
	}

	filePath := filepath.Join(dir, "README.md")

	return b.GenerateFileFromTemplate(ctx, GetTemplate, "README.md",
		filePath, data, 0644)
}
