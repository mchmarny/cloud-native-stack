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
	return b.makeFromRecipeResult(ctx, input, dir)
}

// makeFromRecipeResult generates the NVSentinel bundle from a RecipeResult with component references.
func (b *Bundler) makeFromRecipeResult(ctx context.Context, input recipe.RecipeInput, dir string) (*result.Result, error) {
	start := time.Now()

	slog.Debug("generating NVSentinel bundle from recipe result",
		"output_dir", dir,
		"namespace", Name,
	)

	// Get component reference for nvsentinel
	componentRef := input.GetComponentRef(Name)
	if componentRef == nil {
		return nil, errors.New(errors.ErrCodeInvalidRequest,
			Name+" component not found in recipe")
	}

	// Get values from embedded file
	values, err := input.GetValuesForComponent(Name)
	if err != nil {
		return nil, errors.Wrap(errors.ErrCodeInternal,
			"failed to get values for nvsentinel", err)
	}

	// Apply user value overrides from --set flags to values map
	if overrides := b.getValueOverrides(); len(overrides) > 0 {
		if applyErr := common.ApplyMapOverrides(values, overrides); applyErr != nil {
			slog.Warn("failed to apply some value overrides to values map",
				"error", applyErr,
				"component", Name)
		}
	}

	// Apply system node selector overrides (for system workloads)
	if nodeSelector := b.Config.SystemNodeSelector(); len(nodeSelector) > 0 {
		common.ApplyNodeSelectorOverrides(values, nodeSelector,
			"global.systemNodeSelector",
		)
	}

	// Apply accelerated node tolerations overrides (for GPU node access)
	if tolerations := b.Config.AcceleratedNodeTolerations(); len(tolerations) > 0 {
		common.ApplyTolerationsOverrides(values, tolerations,
			"global.tolerations",
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
		ComponentName:  "NVSentinel",
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

	// Generate bundle metadata (for README and manifest templates)
	metadata := GenerateBundleMetadata(configMap)

	// Create combined data for README (values map + metadata)
	readmeData := map[string]interface{}{
		"Values": values,
		"Script": metadata, // "Script" key preserved for template compatibility
	}

	// Generate README using values map directly
	if b.Config.IncludeReadme() {
		if err := b.generateReadmeFromData(ctx, readmeData, dirs.Root); err != nil {
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

// generateReadmeFromData generates README from pre-built data (for RecipeResult).
func (b *Bundler) generateReadmeFromData(ctx context.Context, data map[string]interface{}, dir string) error {
	filePath := filepath.Join(dir, "README.md")
	return b.GenerateFileFromTemplate(ctx, GetTemplate, "README.md",
		filePath, data, 0644)
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
