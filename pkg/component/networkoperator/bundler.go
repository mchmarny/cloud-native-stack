package networkoperator

import (
	"context"
	"fmt"
	"log/slog"
	"path/filepath"
	"time"

	"github.com/NVIDIA/cloud-native-stack/pkg/bundler/config"
	"github.com/NVIDIA/cloud-native-stack/pkg/bundler/result"
	"github.com/NVIDIA/cloud-native-stack/pkg/bundler/types"
	"github.com/NVIDIA/cloud-native-stack/pkg/component/internal"
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
func (b *Bundler) Make(ctx context.Context, input recipe.RecipeInput, outputDir string) (*result.Result, error) {
	if err := ctx.Err(); err != nil {
		return nil, errors.Wrap(errors.ErrCodeTimeout, "context cancelled", err)
	}

	// For RecipeResult with component references, use values file directly
	if recipe.HasComponentRefs(input) {
		return b.makeFromRecipeResult(ctx, input, outputDir)
	}

	// For legacy Recipe, use measurement-based logic
	r, ok := input.(*recipe.Recipe)
	if !ok {
		return nil, errors.New(errors.ErrCodeInvalidRequest, "unsupported recipe input type")
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

// makeFromRecipeResult generates the Network Operator bundle from a RecipeResult with component references.
func (b *Bundler) makeFromRecipeResult(ctx context.Context, input recipe.RecipeInput, outputDir string) (*result.Result, error) {
	start := time.Now()

	slog.Debug("generating Network Operator bundle from recipe result",
		"output_dir", outputDir,
		"namespace", Name,
	)

	// Get component reference for network-operator
	componentRef := input.GetComponentRef("network-operator")
	if componentRef == nil {
		return nil, errors.New(errors.ErrCodeInvalidRequest,
			"network-operator component not found in recipe")
	}

	// Get values from embedded file
	values, err := input.GetValuesForComponent("network-operator")
	if err != nil {
		return nil, errors.Wrap(errors.ErrCodeInternal,
			"failed to get values for network-operator", err)
	}

	// Create bundle directory structure
	dirs, err := b.CreateBundleDir(outputDir, Name)
	if err != nil {
		return b.Result, errors.Wrap(errors.ErrCodeInternal,
			"failed to create bundle directory", err)
	}

	// Serialize values to YAML
	valuesYAML, err := internal.MarshalYAML(values)
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

	// Generate install/uninstall scripts
	if b.Config.IncludeScripts() {
		if err := b.generateScriptsFromData(ctx, scriptData, dirs.Root); err != nil {
			return b.Result, err
		}
	}

	// Note: For RecipeResult inputs, we only generate values.yaml and checksums
	// README and scripts require legacy Recipe with measurements
	// Users should refer to values.yaml for configuration details

	// Generate checksums file
	if b.Config.IncludeChecksums() {
		if err := b.GenerateChecksums(ctx, dirs.Root); err != nil {
			return b.Result, errors.Wrap(errors.ErrCodeInternal,
				"failed to generate checksums", err)
		}
	}

	// Finalize bundle generation
	b.Finalize(start)

	slog.Debug("Network Operator bundle generated from recipe result",
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

	// Add recipe version from recipe metadata
	if recipeVersion, ok := r.Metadata["recipe-version"]; ok {
		configMap["recipe-version"] = recipeVersion
	}

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

				// Extract registry information from 'registry' subtype
				if st.Name == "registry" {
					// Driver registry URI
					if val, ok := st.Data["uri"]; ok {
						if s, ok := val.Any().(string); ok {
							configMap["driver_registry"] = s
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
	// Get value overrides
	overrides := b.getValueOverrides()

	values := GenerateHelmValues(r, configMap, overrides)
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
	// Get value overrides
	overrides := b.getValueOverrides()

	scriptData := GenerateScriptData(r, configMap)
	helmValues := GenerateHelmValues(r, configMap, overrides)

	data := map[string]interface{}{
		"Script": scriptData,
		"Helm":   helmValues,
	}

	path := filepath.Join(outputDir, "README.md")
	return b.GenerateFileFromTemplate(ctx, GetTemplate, "README.md", path, data, 0644)
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

// getValueOverrides extracts value overrides for this bundler from config.
func (b *Bundler) getValueOverrides() map[string]string {
	allOverrides := b.Config.ValueOverrides()

	// Check both "networkoperator" and "network-operator" keys
	if overrides, ok := allOverrides["networkoperator"]; ok {
		return overrides
	}
	if overrides, ok := allOverrides["network-operator"]; ok {
		return overrides
	}

	return nil
}
