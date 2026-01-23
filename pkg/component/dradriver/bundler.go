package dradriver

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
	"github.com/NVIDIA/cloud-native-stack/pkg/recipe"
)

const (
	Name = "dra-driver"
)

// Bundler creates Nvidia DRA Driver bundles based on recipes.
type Bundler struct {
	*common.BaseBundler
}

// NewBundler creates a new Nvidia DRA Driver bundler instance.
func NewBundler(conf *config.Config) *Bundler {
	return &Bundler{
		BaseBundler: common.NewBaseBundler(conf, types.BundleTypeDraDriver),
	}
}

// Make generates the NVIDIA k8s DRA Driver bundle based on the provided recipe.
// Expects RecipeResult with component references and values maps.
func (b *Bundler) Make(ctx context.Context, input recipe.RecipeInput, dir string) (*result.Result, error) {
	start := time.Now()

	slog.Debug("generating NVIDIA DRA Driver bundle",
		"output_dir", dir,
		"namespace", Name,
	)

	// Get component reference for dra-driver
	componentRef := input.GetComponentRef(Name)
	if componentRef == nil {
		return nil, errors.New(errors.ErrCodeInvalidRequest,
			Name+" component not found in recipe")
	}

	// Get values from component reference
	values, err := input.GetValuesForComponent(Name)
	if err != nil {
		return nil, errors.Wrap(errors.ErrCodeInternal,
			"failed to get values for dra-driver", err)
	}

	// Apply user value overrides from --set flags to values map
	if overrides := b.getValueOverrides(); len(overrides) > 0 {
		if applyErr := common.ApplyMapOverrides(values, overrides); applyErr != nil {
			slog.Warn("failed to apply some value overrides to values map", "error", applyErr)
		}
	}

	// Apply system node tolerations (for dra-driver controller)
	if tolerations := b.Config.SystemNodeTolerations(); len(tolerations) > 0 {
		common.ApplyTolerationsOverrides(values, tolerations,
			"controller.tolerations",
		)
	}

	// Apply accelerated node tolerations (for dra-driver kubelet plugins)
	if tolerations := b.Config.AcceleratedNodeTolerations(); len(tolerations) > 0 {
		common.ApplyTolerationsOverrides(values, tolerations,
			"kubeletPlugin.tolerations",
		)
	}

	// Create bundle directory structure
	dirs, err := b.CreateBundleDir(dir, Name)
	if err != nil {
		return b.Result, errors.Wrap(errors.ErrCodeInternal,
			"failed to create bundle directory", err)
	}

	// Build config map with base settings for metadata extraction
	configMap := b.BuildConfigMapFromInput(input)
	configMap["namespace"] = Name
	configMap["helm_repository"] = componentRef.Source
	configMap["helm_chart_version"] = componentRef.Version

	// Serialize values to YAML with header
	header := common.ValuesHeader{
		ComponentName:  "NVIDIA k8s DRA Driver",
		Timestamp:      time.Now().Format(time.RFC3339),
		BundlerVersion: configMap["bundler_version"],
		RecipeVersion:  configMap["recipe_version"],
	}
	valuesYAML, err := common.MarshalYAMLWithHeader(values, header)
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

	// Generate ScriptData (metadata only - not in Helm values)
	scriptData := GenerateScriptDataFromConfig(configMap)

	// Create combined data for README (values map + metadata)
	readmeData := map[string]interface{}{
		"Values": values,
		"Script": scriptData,
	}

	// Generate README using values map directly
	if b.Config.IncludeReadme() {
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

	slog.Debug("Nvidia DRA Driver bundle generated from recipe result",
		"files", len(b.Result.Files),
		"size_bytes", b.Result.Size,
		"duration", b.Result.Duration.Round(time.Millisecond),
	)

	return b.Result, nil
}

// generateReadmeFromData generates README from pre-built data.
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

// getValueOverrides retrieves value overrides for this bundler from config.
func (b *Bundler) getValueOverrides() map[string]string {
	allOverrides := b.Config.ValueOverrides()
	if allOverrides == nil {
		return nil
	}
	// Return overrides for "dra-driver" or "dradriver"
	if overrides, ok := allOverrides["dra-driver"]; ok {
		return overrides
	}
	if overrides, ok := allOverrides["dradriver"]; ok {
		return overrides
	}
	return nil
}
