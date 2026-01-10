package nvsentinel

import (
	"context"

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
	Name = "nvsentinel"
)

// Bundler creates NVSentinel application bundles based on recipes.
type Bundler struct {
	*common.BaseBundler
}

// NewBundler creates a new NVSentinel bundler instance.
func NewBundler(conf *config.Config) *Bundler {
	return &Bundler{
		BaseBundler: common.NewBaseBundler(conf, types.BundleTypeNVSentinel),
	}
}

// Make generates the NVSentinel bundle based on the provided recipe.
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
			"K8s measurements are required for NVSentinel bundling", err)
	}

	start := time.Now()

	slog.Debug("generating NVSentinel bundle",
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

	slog.Debug("NVSentinel bundle generated",
		"files", len(b.Result.Files),
		"size_bytes", b.Result.Size,
		"duration", b.Result.Duration.Round(time.Millisecond),
	)

	return b.Result, nil
}

// makeFromRecipeResult generates the NVSentinel bundle from a RecipeResult with component references.
func (b *Bundler) makeFromRecipeResult(ctx context.Context, input recipe.RecipeInput, dir string) (*result.Result, error) {
	start := time.Now()

	slog.Debug("generating NVSentinel bundle from recipe result",
		"output_dir", dir,
		"namespace", Name,
	)

	// Get component reference for nvsentinel
	componentRef := input.GetComponentRef("nvsentinel")
	if componentRef == nil {
		return nil, errors.New(errors.ErrCodeInvalidRequest,
			"nvsentinel component not found in recipe")
	}

	// Get values from embedded file
	values, err := input.GetValuesForComponent("nvsentinel")
	if err != nil {
		return nil, errors.Wrap(errors.ErrCodeInternal,
			"failed to get values for nvsentinel", err)
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
	configMap["nvsentinel_version"] = componentRef.Version
	configMap["nvsentinel_chart_url"] = componentRef.Source

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

	// Generate checksums file
	if b.Config.IncludeChecksums() {
		if err := b.GenerateChecksums(ctx, dirs.Root); err != nil {
			return b.Result, errors.Wrap(errors.ErrCodeInternal,
				"failed to generate checksums", err)
		}
	}

	// Finalize bundle generation
	b.Finalize(start)

	slog.Debug("NVSentinel bundle generated from recipe result",
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
		case measurement.TypeGPU:
			// GPU measurements not used by NVSentinel
		case measurement.TypeOS:
			// OS measurements not used by NVSentinel
		case measurement.TypeSystemD:
			// SystemD measurements not used by NVSentinel
		}
	}

	return configMap
}

// extractK8sConfig extracts Kubernetes-related configuration.
func (b *Bundler) extractK8sConfig(m *measurement.Measurement, configMap map[string]string) {
	for _, st := range m.Subtypes {
		if st.Name == "image" {
			// NVSentinel version
			if val, ok := st.Data["nvsentinel"]; ok {
				if s, ok := val.Any().(string); ok {
					configMap["nvsentinel_version"] = s
				}
			}
		}

		if st.Name == "nvsentinel-config" {
			// Helm chart repository
			if val, ok := st.Data["helm_chart_repo"]; ok {
				if s, ok := val.Any().(string); ok {
					configMap["helm_chart_repo"] = s
				}
			}
			// Helm release name
			if val, ok := st.Data["helm_release_name"]; ok {
				if s, ok := val.Any().(string); ok {
					configMap["helm_release_name"] = s
				}
			}
		}
	}
}

// generateHelmValues generates Helm values file.
func (b *Bundler) generateHelmValues(ctx context.Context, r *recipe.Recipe,
	bundleDir string, config map[string]string) error {
	// Get value overrides
	overrides := b.getValueOverrides()

	helmValues := GenerateHelmValues(r, config, overrides)

	filePath := filepath.Join(bundleDir, "values.yaml")
	return b.GenerateFileFromTemplate(ctx, GetTemplate, "values.yaml",
		filePath, helmValues, 0644)
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
	// Get value overrides
	overrides := b.getValueOverrides()

	helmValues := GenerateHelmValues(recipe, config, overrides)
	scriptData := GenerateScriptData(recipe, config)

	// Combine data structures for README
	data := map[string]interface{}{
		"Helm":     helmValues,
		"Script":   scriptData,
		"Metadata": recipe.Metadata,
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

// getValueOverrides extracts value overrides for this bundler from config.
func (b *Bundler) getValueOverrides() map[string]string {
	allOverrides := b.Config.ValueOverrides()

	// Check both "nvsentinel" and "nv-sentinel" keys
	if overrides, ok := allOverrides["nvsentinel"]; ok {
		return overrides
	}
	if overrides, ok := allOverrides["nv-sentinel"]; ok {
		return overrides
	}

	return nil
}
