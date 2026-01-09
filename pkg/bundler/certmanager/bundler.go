package certmanager

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
	Name = "cert-manager"
)

// Bundler generates cert-manager deployment bundles.
type Bundler struct {
	*internal.BaseBundler
}

// NewBundler creates a new cert-manager bundler.
func NewBundler(cfg *config.Config) *Bundler {
	return &Bundler{
		BaseBundler: internal.NewBaseBundler(cfg, types.BundleTypeCertManager),
	}
}

// Make generates a cert-manager bundle from a recipe.
func (b *Bundler) Make(ctx context.Context, r *recipe.Recipe, outputDir string) (*result.Result, error) {
	if err := ctx.Err(); err != nil {
		return nil, errors.Wrap(errors.ErrCodeTimeout, "context cancelled", err)
	}

	// Validate recipe (cert-manager doesn't strictly require K8s measurements, but it's recommended)
	if err := b.validateRecipe(r); err != nil {
		slog.Warn("recipe validation warning", "warning", err)
	}

	start := time.Now()

	slog.Debug("generating cert-manager bundle",
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

	slog.Debug("cert-manager bundle generated successfully",
		"output_dir", outputDir,
		"files", len(b.Result.Files),
		"duration", time.Since(start),
	)

	return b.Result, nil
}

// validateRecipe checks if the recipe contains recommended measurements.
func (b *Bundler) validateRecipe(r *recipe.Recipe) error {
	// cert-manager can be deployed without specific measurements,
	// but K8s measurements are recommended for version-specific configurations
	if len(r.Measurements) == 0 {
		return fmt.Errorf("recipe contains no measurements")
	}
	return nil
}

// buildConfigMap extracts configuration from the recipe.
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
					// Extract cert-manager version
					if val, ok := st.Data["cert-manager"]; ok {
						if s, ok := val.Any().(string); ok {
							configMap["cert_manager_version"] = s
						}
					}
				}
				if st.Name == "config" {
					// Extract configuration settings
					if val, ok := st.Data["install-crds"]; ok {
						if s, ok := val.Any().(string); ok {
							configMap["install_crds"] = s
						}
					}
					if val, ok := st.Data["enable-webhook"]; ok {
						if s, ok := val.Any().(string); ok {
							configMap["enable_webhook"] = s
						}
					}
					if val, ok := st.Data["replica-count"]; ok {
						if s, ok := val.Any().(string); ok {
							configMap["replica_count"] = s
						}
					}
				}
			}
		case measurement.TypeGPU, measurement.TypeOS, measurement.TypeSystemD:
			// Not used by cert-manager bundler
		}
	}

	return configMap
}

// generateHelmValues generates the Helm values.yaml file.
func (b *Bundler) generateHelmValues(ctx context.Context, r *recipe.Recipe,
	configMap map[string]string, outputDir string) error {
	// Get value overrides
	overrides := b.getValueOverrides()

	helmValues := GenerateHelmValues(r, configMap, overrides)

	filePath := filepath.Join(outputDir, "values.yaml")
	return b.GenerateFileFromTemplate(ctx, GetTemplate, "values.yaml",
		filePath, helmValues, 0644)
}

// generateScripts generates installation and uninstallation scripts.
func (b *Bundler) generateScripts(ctx context.Context, r *recipe.Recipe,
	configMap map[string]string, scriptsDir string) error {

	scriptData := GenerateScriptData(r, configMap)

	// Generate install script
	installPath := filepath.Join(scriptsDir, "install.sh")
	if err := b.GenerateFileFromTemplate(ctx, GetTemplate, "install.sh",
		installPath, scriptData, 0755); err != nil {
		return err
	}

	// Generate uninstall script
	uninstallPath := filepath.Join(scriptsDir, "uninstall.sh")
	return b.GenerateFileFromTemplate(ctx, GetTemplate, "uninstall.sh",
		uninstallPath, scriptData, 0755)
}

// generateReadme generates the README documentation.
func (b *Bundler) generateReadme(ctx context.Context, r *recipe.Recipe,
	configMap map[string]string, outputDir string) error {
	// Get value overrides
	overrides := b.getValueOverrides()

	// Combine helm values and script data for README
	readmeData := map[string]interface{}{
		"Helm":   GenerateHelmValues(r, configMap, overrides),
		"Script": GenerateScriptData(r, configMap),
	}

	filePath := filepath.Join(outputDir, "README.md")
	return b.GenerateFileFromTemplate(ctx, GetTemplate, "README.md",
		filePath, readmeData, 0644)
}

// getValueOverrides extracts value overrides for this bundler from config.
func (b *Bundler) getValueOverrides() map[string]string {
	allOverrides := b.Config.ValueOverrides()

	// Check both "certmanager" and "cert-manager" keys
	if overrides, ok := allOverrides["certmanager"]; ok {
		return overrides
	}
	if overrides, ok := allOverrides["cert-manager"]; ok {
		return overrides
	}

	return nil
}
