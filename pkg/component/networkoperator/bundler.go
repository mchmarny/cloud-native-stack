package networkoperator

import (
	"context"
	"log/slog"
	"path/filepath"
	"time"

	"github.com/NVIDIA/cloud-native-stack/pkg/bundler/config"
	"github.com/NVIDIA/cloud-native-stack/pkg/bundler/result"
	"github.com/NVIDIA/cloud-native-stack/pkg/bundler/types"
	"github.com/NVIDIA/cloud-native-stack/pkg/component/internal"
	"github.com/NVIDIA/cloud-native-stack/pkg/errors"
	"github.com/NVIDIA/cloud-native-stack/pkg/recipe"
)

const (
	Name = "network-operator"
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

	return b.makeFromRecipeResult(ctx, input, outputDir)
}

// makeFromRecipeResult generates the Network Operator bundle from a RecipeResult with component references.
func (b *Bundler) makeFromRecipeResult(ctx context.Context, input recipe.RecipeInput, outputDir string) (*result.Result, error) {
	start := time.Now()

	slog.Debug("generating Network Operator bundle from recipe result",
		"output_dir", outputDir,
		"namespace", Name,
	)

	// Get component reference for network-operator
	componentRef := input.GetComponentRef(Name)
	if componentRef == nil {
		return nil, errors.New(errors.ErrCodeInvalidRequest,
			Name+" component not found in recipe")
	}

	// Get values from embedded file
	values, err := input.GetValuesForComponent(Name)
	if err != nil {
		return nil, errors.Wrap(errors.ErrCodeInternal,
			"failed to get values for network-operator", err)
	}

	// Apply user value overrides from --set flags to values map
	if overrides := b.getValueOverrides(); len(overrides) > 0 {
		if applyErr := internal.ApplyMapOverrides(values, overrides); applyErr != nil {
			slog.Warn("failed to apply some value overrides to values map", "error", applyErr)
		}
	}

	// Create bundle directory structure
	dirs, err := b.CreateBundleDir(outputDir, Name)
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
	header := internal.ValuesHeader{
		ComponentName:  "Network Operator",
		BundlerVersion: configMap["bundler_version"],
		RecipeVersion:  configMap["recipe_version"],
	}
	valuesYAML, err := internal.MarshalYAMLWithHeader(values, header)
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

// generateReadmeFromData generates README from pre-built data (for RecipeResult).
func (b *Bundler) generateReadmeFromData(ctx context.Context, data map[string]interface{}, dir string) error {
	filePath := filepath.Join(dir, "README.md")
	return b.GenerateFileFromTemplate(ctx, GetTemplate, "README.md",
		filePath, data, 0644)
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
