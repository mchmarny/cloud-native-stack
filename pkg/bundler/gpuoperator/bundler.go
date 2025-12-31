package gpuoperator

import (
	"context"
	"fmt"
	"log/slog"
	"path/filepath"
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
	Name = "gpu-operator"

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
func (b *Bundler) Make(ctx context.Context, r *recipe.Recipe, dir string) (*result.Result, error) {
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

// buildConfigMap extracts configuration from recipe and bundler config.
func (b *Bundler) buildConfigMap(r *recipe.Recipe) map[string]string {
	// Start with base config (namespace, helm settings, labels, annotations)
	configMap := b.BuildBaseConfigMap()
	configMap["namespace"] = Name

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

				// Extract configuration flags from 'config' subtype
				if st.Name == configSubtype {
					// MIG configuration
					if val, ok := st.Data["mig"]; ok {
						if b, ok := val.Any().(bool); ok {
							configMap["enable_mig"] = fmt.Sprintf("%t", b)
							// If MIG is enabled, use mixed strategy
							if b {
								configMap["mig_strategy"] = "mixed"
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

	helmValues := GenerateHelmValues(r, config)

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

	manifestData := GenerateManifestData(r, config)
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

	helmValues := GenerateHelmValues(recipe, config)
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
