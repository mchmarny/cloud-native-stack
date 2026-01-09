package nvsentinel

import (
	"context"

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
func (b *Bundler) Make(ctx context.Context, r *recipe.Recipe, dir string) (*result.Result, error) {
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

		if st.Name == "registry" {
			// Registry URI
			if val, ok := st.Data["uri"]; ok {
				if s, ok := val.Any().(string); ok {
					configMap["helm_chart_repo"] = s
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
