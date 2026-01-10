package gpuoperator

import (
	"context"
	"fmt"
	"log/slog"
	"path/filepath"
	"time"

	"github.com/NVIDIA/cloud-native-stack/pkg/bundler/config"
	"github.com/NVIDIA/cloud-native-stack/pkg/bundler/result"
	"github.com/NVIDIA/cloud-native-stack/pkg/bundler/types"
	common "github.com/NVIDIA/cloud-native-stack/pkg/component/internal"
	"github.com/NVIDIA/cloud-native-stack/pkg/errors"
	"github.com/NVIDIA/cloud-native-stack/pkg/measurement"
	"github.com/NVIDIA/cloud-native-stack/pkg/recipe"
)

const (
	Name          = "gpu-operator"
	strMixed      = "mixed"
	configSubtype = "config"
)

// Bundler creates GPU Operator application bundles based on recipes.
type Bundler struct {
	*common.BaseBundler
}

// NewBundler creates a new GPU Operator bundler instance.
func NewBundler(conf *config.Config) *Bundler {
	return &Bundler{
		BaseBundler: common.NewBaseBundler(conf, types.BundleTypeGpuOperator),
	}
}

// Make generates the GPU Operator bundle based on the provided recipe.
func (b *Bundler) Make(ctx context.Context, input recipe.RecipeInput, dir string) (*result.Result, error) {
	// For RecipeResult with component references, use values file directly
	if recipe.HasComponentRefs(input) {
		return b.makeFromRecipeResult(ctx, input, dir)
	}

	// For legacy Recipe, use measurement-based logic
	r, ok := input.(*recipe.Recipe)
	if !ok {
		return nil, errors.New(errors.ErrCodeInvalidRequest, "unsupported recipe input type")
	}

	// Check for required measurements
	if err := r.ValidateMeasurementExists(measurement.TypeK8s); err != nil {
		return nil, errors.Wrap(errors.ErrCodeInvalidRequest,
			"measurements are required for GPU Operator bundling", err)
	}

	// Check for GPU measurements (optional but recommended)
	if err := r.ValidateMeasurementExists(measurement.TypeGPU); err != nil {
		slog.Warn("GPU measurements not found in recipe", "warning", err)
	}

	start := time.Now()

	slog.Debug("generating GPU Operator bundle",
		"output_dir", dir,
		"namespace", Name,
	)

	// Create bundle directory structure
	dirs, err := b.CreateBundleDir(dir, Name)
	if err != nil {
		return b.Result, errors.Wrap(errors.ErrCodeInternal,
			"failed to create bundle directory", err)
	}

	// Prepare configuration map and override values for this bundler
	configMap := b.buildConfigMap(r)

	// Generate Helm values
	if err := b.generateHelmValues(ctx, r, dirs.Root, configMap); err != nil {
		return b.Result, err
	}

	// Generate ClusterPolicy manifest
	if err := b.generateClusterPolicy(ctx, r, dirs.Manifests, configMap); err != nil {
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

	slog.Debug("GPU Operator bundle generated",
		"files", len(b.Result.Files),
		"size_bytes", b.Result.Size,
		"duration", b.Result.Duration.Round(time.Millisecond),
	)

	return b.Result, nil
}

// makeFromRecipeResult generates the GPU Operator bundle from a RecipeResult with component references.
// RecipeResults provide pre-computed values files from the components directory.
func (b *Bundler) makeFromRecipeResult(ctx context.Context, input recipe.RecipeInput, dir string) (*result.Result, error) {
	start := time.Now()

	slog.Debug("generating GPU Operator bundle from recipe result",
		"output_dir", dir,
		"namespace", Name,
	)

	// Get component reference for gpu-operator
	componentRef := input.GetComponentRef("gpu-operator")
	if componentRef == nil {
		return nil, errors.New(errors.ErrCodeInvalidRequest,
			"gpu-operator component not found in recipe")
	}

	// Get values from embedded file
	values, err := input.GetValuesForComponent("gpu-operator")
	if err != nil {
		return nil, errors.Wrap(errors.ErrCodeInternal,
			"failed to get values for gpu-operator", err)
	}

	// Apply user value overrides from --set flags to values map
	if overrides := b.getValueOverrides(); len(overrides) > 0 {
		if applyErr := common.ApplyMapOverrides(values, overrides); applyErr != nil {
			slog.Warn("failed to apply some value overrides to values map", "error", applyErr)
		}
	}

	// Create bundle directory structure
	dirs, err := b.CreateBundleDir(dir, Name)
	if err != nil {
		return b.Result, errors.Wrap(errors.ErrCodeInternal,
			"failed to create bundle directory", err)
	}

	// Serialize values to YAML
	valuesYAML, err := common.MarshalYAML(values)
	if err != nil {
		return b.Result, errors.Wrap(errors.ErrCodeInternal,
			"failed to serialize values to YAML", err)
	}

	// Write values.yaml
	valuesPath := filepath.Join(dirs.Root, "values.yaml")
	if err := b.WriteFile(valuesPath, valuesYAML, 0644); err != nil {
		return b.Result, errors.Wrap(errors.ErrCodeInternal,
			"failed to write values file", err)
	}

	// Build config map with base settings and recipe version
	configMap := b.BuildConfigMapFromInput(input)
	configMap["namespace"] = Name
	configMap["helm_repository"] = componentRef.Source
	configMap["helm_chart_version"] = componentRef.Version

	// Generate HelmValues from values map
	helmValues := GenerateHelmValuesFromMap(configMap)

	// Apply user value overrides from --set flags
	helmValues.applyValueOverrides(b.getValueOverrides())

	// Generate ScriptData from config
	scriptData := GenerateScriptDataFromConfig(configMap)

	// Generate README
	if b.Config.IncludeReadme() {
		readmeData := map[string]interface{}{
			"Helm":   helmValues,
			"Script": scriptData,
		}
		if err := b.generateReadmeFromData(ctx, readmeData, dirs.Root); err != nil {
			return b.Result, err
		}
	}

	// Generate ClusterPolicy manifest
	if err := b.generateManifestsFromData(ctx, helmValues, dirs.Root); err != nil {
		return b.Result, err
	}

	// Generate install/uninstall scripts
	if b.Config.IncludeScripts() {
		if err := b.generateScriptsFromData(ctx, scriptData, dirs.Root); err != nil {
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

	slog.Debug("GPU Operator bundle generated from recipe result",
		"files", len(b.Result.Files),
		"size_bytes", b.Result.Size,
		"duration", b.Result.Duration.Round(time.Millisecond),
	)

	return b.Result, nil
}

// buildConfigMap extracts configuration from recipe and bundler config.
func (b *Bundler) buildConfigMap(r *recipe.Recipe) map[string]string {
	// Start with base config (namespace, helm settings, labels, annotations)
	configMap := b.BuildBaseConfigMap()
	configMap["namespace"] = Name

	// Add recipe version from recipe metadata
	if recipeVersion, ok := r.Metadata["recipe-version"]; ok {
		configMap["recipe-version"] = recipeVersion
	}

	// Extract values from recipe measurements
	for _, m := range r.Measurements {
		switch m.Type {
		case measurement.TypeK8s:
			for _, st := range m.Subtypes {
				// Extract image versions from 'image' subtype
				if st.Name == "image" {
					// GPU Operator version
					if val, ok := st.Data["gpu-operator"]; ok {
						if s, ok := val.Any().(string); ok {
							configMap["gpu_operator_version"] = s
						}
					}
					// Driver version
					if val, ok := st.Data["driver"]; ok {
						if s, ok := val.Any().(string); ok {
							configMap["driver_version"] = s
						}
					}
					// Container Toolkit version
					if val, ok := st.Data["container-toolkit"]; ok {
						if s, ok := val.Any().(string); ok {
							configMap["container_toolkit_version"] = s
						}
					}
					// Device Plugin version
					if val, ok := st.Data["k8s-device-plugin"]; ok {
						if s, ok := val.Any().(string); ok {
							configMap["device_plugin_version"] = s
						}
					}
					// DCGM version
					if val, ok := st.Data["dcgm"]; ok {
						if s, ok := val.Any().(string); ok {
							configMap["dcgm_version"] = s
						}
					}
					// DCGM Exporter version
					if val, ok := st.Data["dcgm-exporter"]; ok {
						if s, ok := val.Any().(string); ok {
							configMap["dcgm_exporter_version"] = s
						}
					}
					// GDRCopy driver version
					if val, ok := st.Data["gdrdrv"]; ok {
						if s, ok := val.Any().(string); ok {
							configMap["gdrdrv_version"] = s
						}
					}
				}

				// Extract registry information from 'registry' subtype
				if st.Name == "registry" {
					// Driver registry URI
					if val, ok := st.Data["uri"]; ok {
						if s, ok := val.Any().(string); ok {
							configMap["driver_registry"] = s
						}
					}
				}

				// Extract configuration flags from 'config' subtype
				if st.Name == configSubtype {
					// MIG configuration
					if val, ok := st.Data["mig"]; ok {
						if b, ok := val.Any().(bool); ok {
							configMap["enable_mig"] = fmt.Sprintf("%t", b)
							// If MIG is enabled, use mixed strategy
							if b {
								configMap["mig_strategy"] = strMixed
							}
						}
					}
					// Open Kernel Module setting
					if val, ok := st.Data["useOpenKernelModule"]; ok {
						if b, ok := val.Any().(bool); ok {
							configMap["use_open_kernel_module"] = fmt.Sprintf("%t", b)
						}
					}
					// RDMA support (affects GDS)
					if val, ok := st.Data["rdma"]; ok {
						if b, ok := val.Any().(bool); ok {
							configMap["enable_gds"] = fmt.Sprintf("%t", b)
						}
					}
					// CDI support
					if val, ok := st.Data["cdi"]; ok {
						if b, ok := val.Any().(bool); ok {
							configMap["enable_cdi"] = fmt.Sprintf("%t", b)
						}
					}
					// vGPU configuration
					if val, ok := st.Data["vgpu"]; ok {
						if b, ok := val.Any().(bool); ok {
							configMap["enable_vgpu"] = fmt.Sprintf("%t", b)
						}
					}
					// vGPU License Server
					if val, ok := st.Data["vgpuLicenseServer"]; ok {
						if s, ok := val.Any().(string); ok && s != "" {
							configMap["vgpu_license_server"] = s
						}
					}
					// Secure Boot
					if val, ok := st.Data["secureBoot"]; ok {
						if b, ok := val.Any().(bool); ok {
							configMap["enable_secure_boot"] = fmt.Sprintf("%t", b)
						}
					}
				}
			}
		case measurement.TypeGPU:
			for _, st := range m.Subtypes {
				// Extract driver version from nvidia-smi output if not already set
				if st.Name == "smi" {
					if _, exists := configMap["driver_version"]; !exists {
						if val, ok := st.Data["driver-version"]; ok {
							if s, ok := val.Any().(string); ok {
								configMap["driver_version"] = s
							}
						}
					}
				}
			}
		case measurement.TypeOS, measurement.TypeSystemD:
			// OS and SystemD measurements not used for GPU Operator configuration
			continue
		default:
			// Other measurement types not used for GPU Operator configuration
			continue
		}
	}

	return configMap
}

// generateHelmValues generates Helm values file.
func (b *Bundler) generateHelmValues(ctx context.Context, r *recipe.Recipe,
	bundleDir string, config map[string]string) error {

	// Get value overrides for this bundler from config
	overrides := b.getValueOverrides()

	helmValues := GenerateHelmValues(r, config, overrides)

	if errValidate := helmValues.Validate(); errValidate != nil {
		return errors.Wrap(errors.ErrCodeInvalidRequest, "invalid helm values", errValidate)
	}

	filePath := filepath.Join(bundleDir, "values.yaml")
	return b.GenerateFileFromTemplate(ctx, GetTemplate, "values.yaml",
		filePath, helmValues, 0644)
}

// generateClusterPolicy generates ClusterPolicy manifest.
func (b *Bundler) generateClusterPolicy(ctx context.Context, r *recipe.Recipe,
	dir string, config map[string]string) error {

	overrides := b.getValueOverrides()
	manifestData := GenerateManifestData(r, config, overrides)
	filePath := filepath.Join(dir, "clusterpolicy.yaml")

	return b.GenerateFileFromTemplate(ctx, GetTemplate, "clusterpolicy",
		filePath, manifestData, 0644)
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

	overrides := b.getValueOverrides()
	helmValues := GenerateHelmValues(recipe, config, overrides)
	scriptData := GenerateScriptData(recipe, config)

	// Combine both data structures for README
	data := map[string]interface{}{
		"Helm":   helmValues,
		"Script": scriptData,
	}

	filePath := filepath.Join(dir, "README.md")

	return b.GenerateFileFromTemplate(ctx, GetTemplate, "README.md",
		filePath, data, 0644)
}

// generateReadmeFromData generates README from pre-built data (for RecipeResult).
func (b *Bundler) generateReadmeFromData(ctx context.Context, data map[string]interface{}, dir string) error {
	filePath := filepath.Join(dir, "README.md")
	return b.GenerateFileFromTemplate(ctx, GetTemplate, "README.md",
		filePath, data, 0644)
}

// generateScriptsFromData generates install/uninstall scripts from pre-built data.
func (b *Bundler) generateScriptsFromData(ctx context.Context, scriptData *ScriptData, dir string) error {
	// Generate install script
	installPath := filepath.Join(dir, "scripts", "install.sh")
	if err := b.GenerateFileFromTemplate(ctx, GetTemplate, "install.sh",
		installPath, scriptData, 0755); err != nil {
		return err
	}

	// Generate uninstall script
	uninstallPath := filepath.Join(dir, "scripts", "uninstall.sh")
	return b.GenerateFileFromTemplate(ctx, GetTemplate, "uninstall.sh",
		uninstallPath, scriptData, 0755)
}

// generateManifestsFromData generates manifests from pre-built data.
func (b *Bundler) generateManifestsFromData(ctx context.Context, helmValues *HelmValues, dir string) error {
	manifestData := GenerateManifestDataFromValues(helmValues)
	filePath := filepath.Join(dir, "manifests", "clusterpolicy.yaml")
	return b.GenerateFileFromTemplate(ctx, GetTemplate, "clusterpolicy",
		filePath, manifestData, 0644)
}

// getValueOverrides retrieves value overrides for this bundler from config.
func (b *Bundler) getValueOverrides() map[string]string {
	allOverrides := b.Config.ValueOverrides()
	if allOverrides == nil {
		return nil
	}
	// Return overrides for "gpuoperator" or "gpu-operator"
	if overrides, ok := allOverrides["gpuoperator"]; ok {
		return overrides
	}
	if overrides, ok := allOverrides["gpu-operator"]; ok {
		return overrides
	}
	return nil
}
