package skyhook

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
	Name = "skyhook-operator"
)

// Bundler creates Skyhook Operator application bundles based on recipes.
type Bundler struct {
	*common.BaseBundler
}

// NewBundler creates a new Skyhook bundler instance.
func NewBundler(conf *config.Config) *Bundler {
	return &Bundler{
		BaseBundler: common.NewBaseBundler(conf, types.BundleTypeSkyhook),
	}
}

// Make generates the Skyhook bundle based on the provided recipe.
func (b *Bundler) Make(ctx context.Context, input recipe.RecipeInput, dir string) (*result.Result, error) {
	return b.makeFromRecipeResult(ctx, input, dir)
}

// makeFromRecipeResult generates the Skyhook bundle from a RecipeResult with component references.
func (b *Bundler) makeFromRecipeResult(ctx context.Context, input recipe.RecipeInput, dir string) (*result.Result, error) {
	start := time.Now()

	slog.Debug("generating Skyhook bundle from recipe result",
		"output_dir", dir,
		"namespace", Name,
	)

	// Get component reference for skyhook-operator
	componentRef := input.GetComponentRef(Name)
	if componentRef == nil {
		return nil, errors.New(errors.ErrCodeInvalidRequest,
			Name+" component not found in recipe")
	}

	// Get values from embedded file
	values, err := input.GetValuesForComponent(Name)
	if err != nil {
		return nil, errors.Wrap(errors.ErrCodeInternal,
			"failed to get values for skyhook", err)
	}

	// Apply user value overrides from --set flags to values map
	if overrides := b.getValueOverrides(); len(overrides) > 0 {
		if applyErr := common.ApplyMapOverrides(values, overrides); applyErr != nil {
			slog.Warn("failed to apply some value overrides to values map",
				"error", applyErr,
				"component", Name)
		}
	}

	// Apply accelerated node selector overrides from CLI flags
	// Skyhook is for GPU/accelerated nodes, so use accelerated node selectors/tolerations
	common.ApplyNodeSelectorOverrides(values, b.Config.AcceleratedNodeSelector(),
		"controllerManager.selectors")

	// Apply accelerated tolerations overrides from CLI flags
	common.ApplyTolerationsOverrides(values, b.Config.AcceleratedNodeTolerations(),
		"controllerManager.tolerations")

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
		ComponentName:  "Skyhook",
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

	// Generate customization manifests if specified in values
	if err := b.generateCustomizationManifests(ctx, values, scriptData, dirs.Root); err != nil {
		return b.Result, err
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

	slog.Debug("Skyhook bundle generated from recipe result",
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

	// Check "skyhook-operator" key (also accept "skyhook" for backward compatibility)
	if overrides, ok := allOverrides["skyhook-operator"]; ok {
		return overrides
	}
	// Backward compatibility: also check "skyhook" key
	if overrides, ok := allOverrides["skyhook"]; ok {
		return overrides
	}

	return nil
}

// generateCustomizationManifests generates Skyhook customization CR manifests based on values.
// If customization is specified in values but doesn't exist, returns an error.
func (b *Bundler) generateCustomizationManifests(ctx context.Context, values map[string]interface{}, scriptData *ScriptData, dir string) error {
	// Check if customization is specified in values
	customizationName, ok := values["customization"].(string)
	if !ok || customizationName == "" {
		// No customization specified, nothing to generate
		return nil
	}

	slog.Debug("generating Skyhook customization manifest",
		"customization", customizationName,
	)

	// Check if the customization template exists
	_, exists := GetCustomizationTemplate(customizationName)
	if !exists {
		availableCustomizations := ListCustomizations()
		return errors.New(errors.ErrCodeInvalidRequest,
			"unknown Skyhook customization '"+customizationName+"'; available customizations: "+
				formatCustomizationList(availableCustomizations))
	}

	// Combine values map with script metadata for template
	manifestData := map[string]interface{}{
		"Values": values,
		"Script": scriptData,
	}

	// Add accelerated node tolerations if provided via CLI flags
	if tolerations := b.Config.AcceleratedNodeTolerations(); len(tolerations) > 0 {
		manifestData["Tolerations"] = common.TolerationsToPodSpec(tolerations)
	}

	// Add accelerated node selectors as matchExpressions if provided via CLI flags
	if nodeSelector := b.Config.AcceleratedNodeSelector(); len(nodeSelector) > 0 {
		manifestData["NodeSelectorExpressions"] = common.NodeSelectorToMatchExpressions(nodeSelector)
	}

	// Generate the customization manifest (WriteFile creates parent dirs automatically)
	filePath := filepath.Join(dir, "manifests", customizationName+".yaml")
	if err := b.GenerateFileFromTemplate(ctx, GetCustomizationTemplate, customizationName,
		filePath, manifestData, 0644); err != nil {
		return errors.Wrap(errors.ErrCodeInternal,
			"failed to generate customization manifest", err)
	}

	return nil
}

// formatCustomizationList formats a list of customization names for error messages.
func formatCustomizationList(names []string) string {
	if len(names) == 0 {
		return "(none available)"
	}
	result := ""
	for i, name := range names {
		if i > 0 {
			result += ", "
		}
		result += name
	}
	return result
}
