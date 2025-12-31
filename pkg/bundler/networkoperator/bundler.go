package networkoperator

import (
	"context"
	"fmt"
	"log/slog"
	"path/filepath"
	"time"

	"github.com/NVIDIA/cloud-native-stack/pkg/bundler/config"
	"github.com/NVIDIA/cloud-native-stack/pkg/bundler/internal"
	"github.com/NVIDIA/cloud-native-stack/pkg/bundler/result"
	"github.com/NVIDIA/cloud-native-stack/pkg/bundler/types"
	"github.com/NVIDIA/cloud-native-stack/pkg/errors"
	"github.com/NVIDIA/cloud-native-stack/pkg/measurement"
	"github.com/NVIDIA/cloud-native-stack/pkg/recipe"
)

const (
	Name = "network-operator"

	configSubtype = "config"
)

// Bundler generates Network Operator deployment bundles.
type Bundler struct {
	*internal.BaseBundler
}

// NewBundler creates a new Network Operator bundler.
func NewBundler(cfg *config.Config) *Bundler {
	return &Bundler{
		BaseBundler: internal.NewBaseBundler(cfg, types.BundleTypeNetworkOperator),
	}
}

// Make generates a Network Operator bundle from a recipe.
func (b *Bundler) Make(ctx context.Context, r *recipe.Recipe, outputDir string) (*result.Result, error) {
	if err := ctx.Err(); err != nil {
		return nil, errors.Wrap(errors.ErrCodeTimeout, "context cancelled", err)
	}

	// Validate recipe has required measurements
	if err := b.validateRecipe(r); err != nil {
		return nil, errors.Wrap(errors.ErrCodeInvalidRequest, "invalid recipe", err)
	}

	start := time.Now()

	slog.Debug("generating Network Operator bundle",
		"output_dir", outputDir,
		"namespace", Name,
	)

	// Create bundle directory structure
	dirs, err := b.CreateBundleDir(outputDir, Name)
	if err != nil {
		return b.Result, errors.Wrap(errors.ErrCodeInternal,
			"failed to create bundle directory", err)
	}

	// Build configuration map from recipe and bundler config
	configMap := b.buildConfigMap(r)

	// Generate all bundle components
	if err := b.generateHelmValues(ctx, r, configMap, dirs.Root); err != nil {
		return b.Result, errors.Wrap(errors.ErrCodeInternal,
			"failed to generate helm values", err)
	}

	if err := b.generateNicClusterPolicy(ctx, r, configMap, dirs.Manifests); err != nil {
		return b.Result, errors.Wrap(errors.ErrCodeInternal,
			"failed to generate NicClusterPolicy", err)
	}

	if b.Config.IncludeScripts() {
		if err := b.generateScripts(ctx, r, configMap, dirs.Scripts); err != nil {
			return b.Result, errors.Wrap(errors.ErrCodeInternal,
				"failed to generate scripts", err)
		}
	}

	if b.Config.IncludeReadme() {
		if err := b.generateReadme(ctx, r, configMap, dirs.Root); err != nil {
			return b.Result, errors.Wrap(errors.ErrCodeInternal,
				"failed to generate README", err)
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

	slog.Debug("Network Operator bundle generated",
		"files", len(b.Result.Files),
		"size_bytes", b.Result.Size,
		"duration", b.Result.Duration.Round(time.Millisecond),
	)

	return b.Result, nil
}

// validateRecipe checks if recipe has required measurements.
func (b *Bundler) validateRecipe(r *recipe.Recipe) error {
	if r == nil {
		return fmt.Errorf("recipe is nil")
	}

	// Check for required K8s measurements
	hasK8s := false
	for _, m := range r.Measurements {
		if m.Type == measurement.TypeK8s {
			hasK8s = true
			break
		}
	}

	if !hasK8s {
		return fmt.Errorf("recipe missing required Kubernetes measurements")
	}

	return nil
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
				if st.Name == "image" {
					// Extract Network Operator version
					if val, ok := st.Data["network-operator"]; ok {
						if s, ok := val.Any().(string); ok {
							configMap["network_operator_version"] = s
						}
					}
					// Extract OFED driver version
					if val, ok := st.Data["ofed-driver"]; ok {
						if s, ok := val.Any().(string); ok {
							configMap["ofed_version"] = s
						}
					}
				}

				if st.Name == configSubtype {
					// Extract RDMA setting
					if val, ok := st.Data["rdma"]; ok {
						if b, ok := val.Any().(bool); ok {
							configMap["enable_rdma"] = fmt.Sprintf("%t", b)
						}
					}
					// Extract SR-IOV setting
					if val, ok := st.Data["sr-iov"]; ok {
						if b, ok := val.Any().(bool); ok {
							configMap["enable_sriov"] = fmt.Sprintf("%t", b)
						}
					}
				}
			}
		case measurement.TypeGPU, measurement.TypeOS, measurement.TypeSystemD:
			// GPU, OS and SystemD measurements not used for Network Operator configuration
			continue
		default:
			// Other types are not used for Network Operator configuration
			continue
		}
	}

	return configMap
}

// generateHelmValues creates the Helm values.yaml file.
func (b *Bundler) generateHelmValues(ctx context.Context, r *recipe.Recipe, configMap map[string]string,
	outputDir string) error {

	values := GenerateHelmValues(r, configMap)
	if err := values.Validate(); err != nil {
		return errors.Wrap(errors.ErrCodeInvalidRequest, "invalid helm values", err)
	}

	path := filepath.Join(outputDir, "values.yaml")
	return b.GenerateFileFromTemplate(ctx, GetTemplate, "values.yaml", path, values, 0644)
}

// generateNicClusterPolicy creates the NicClusterPolicy manifest.
func (b *Bundler) generateNicClusterPolicy(ctx context.Context, r *recipe.Recipe, configMap map[string]string,
	manifestsDir string) error {

	data := GenerateManifestData(r, configMap)
	path := filepath.Join(manifestsDir, "nicclusterpolicy.yaml")
	return b.GenerateFileFromTemplate(ctx, GetTemplate, "nicclusterpolicy", path, data, 0644)
}

// generateScripts creates installation and uninstallation scripts.
func (b *Bundler) generateScripts(ctx context.Context, r *recipe.Recipe, configMap map[string]string,
	scriptsDir string) error {

	scriptData := GenerateScriptData(r, configMap)

	// Generate install script
	installPath := filepath.Join(scriptsDir, "install.sh")
	if err := b.GenerateFileFromTemplate(ctx, GetTemplate, "install.sh", installPath, scriptData, 0755); err != nil {
		return err
	}

	// Generate uninstall script
	uninstallPath := filepath.Join(scriptsDir, "uninstall.sh")
	return b.GenerateFileFromTemplate(ctx, GetTemplate, "uninstall.sh", uninstallPath, scriptData, 0755)
}

// generateReadme creates the README.md file.
func (b *Bundler) generateReadme(ctx context.Context, r *recipe.Recipe, configMap map[string]string,
	outputDir string) error {

	scriptData := GenerateScriptData(r, configMap)
	helmValues := GenerateHelmValues(r, configMap)

	data := map[string]interface{}{
		"Script": scriptData,
		"Helm":   helmValues,
	}

	path := filepath.Join(outputDir, "README.md")
	return b.GenerateFileFromTemplate(ctx, GetTemplate, "README.md", path, data, 0644)
}
