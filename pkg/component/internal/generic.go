package internal

import (
	"context"
	"log/slog"
	"path/filepath"
	"time"

	"github.com/NVIDIA/cloud-native-stack/pkg/bundler/result"
	"github.com/NVIDIA/cloud-native-stack/pkg/errors"
	"github.com/NVIDIA/cloud-native-stack/pkg/recipe"
)

// ComponentConfig defines the configuration for a bundler component.
// This struct captures all component-specific settings, allowing the generic
// MakeBundle function to handle the common bundling logic.
type ComponentConfig struct {
	// Name is the component identifier used in recipes (e.g., "gpu-operator").
	Name string

	// DisplayName is the human-readable name used in templates (e.g., "GPU Operator").
	DisplayName string

	// ValueOverrideKeys are alternative keys to check for value overrides.
	// The Name is always checked first, then these alternatives (e.g., ["gpuoperator"]).
	ValueOverrideKeys []string

	// SystemNodeSelectorPaths are Helm value paths for system component node selectors.
	// Example: ["operator.nodeSelector", "nfd.nodeSelector"]
	SystemNodeSelectorPaths []string

	// SystemTolerationPaths are Helm value paths for system component tolerations.
	// Example: ["operator.tolerations"]
	SystemTolerationPaths []string

	// AcceleratedNodeSelectorPaths are Helm value paths for GPU node selectors.
	// Example: ["daemonsets.nodeSelector"]
	AcceleratedNodeSelectorPaths []string

	// AcceleratedTolerationPaths are Helm value paths for GPU node tolerations.
	// Example: ["daemonsets.tolerations"]
	AcceleratedTolerationPaths []string

	// DefaultHelmRepository is the default Helm repository URL.
	DefaultHelmRepository string

	// DefaultHelmChart is the chart name (e.g., "nvidia/gpu-operator").
	DefaultHelmChart string

	// DefaultHelmChartVersion is the default chart version if not specified in recipe.
	DefaultHelmChartVersion string

	// TemplateGetter is the function that retrieves templates by name.
	// If nil, TemplateNames will be used with the embedded TemplatesFS.
	TemplateGetter TemplateFunc

	// TemplateNames lists the template files to embed (e.g., ["README.md"]).
	// Used with auto-generated template getter when TemplateGetter is nil.
	TemplateNames []string

	// CustomManifestFunc is an optional function to generate additional manifests.
	// It receives the values map, config map, and output directory.
	// It should return the list of generated file paths, or nil if no manifests were generated.
	CustomManifestFunc CustomManifestFunc

	// MetadataFunc creates component-specific metadata for templates.
	// If nil, the default BundleMetadata is used.
	MetadataFunc MetadataFunc

	// MetadataExtensions provides additional fields for BundleMetadata.
	// These are merged into the Extensions map of the generated metadata.
	// Use this instead of MetadataFunc for simple extensions.
	MetadataExtensions map[string]interface{}
}

// CustomManifestFunc is a function type for generating custom manifests.
// It receives context, base bundler, values map, config map, and output directory.
// Returns slice of generated file paths (may be nil/empty if no manifests needed).
type CustomManifestFunc func(ctx context.Context, b *BaseBundler, values map[string]interface{}, configMap map[string]string, dir string) ([]string, error)

// MetadataFunc is a function type for creating component-specific metadata.
type MetadataFunc func(configMap map[string]string) interface{}

// BundleMetadata contains common metadata used for README and manifest template rendering.
// This is the default metadata structure used when MetadataFunc is not provided.
// The Extensions map allows component-specific fields without custom structs.
type BundleMetadata struct {
	// Common fields used by all components
	Namespace        string
	HelmRepository   string
	HelmChart        string
	HelmChartVersion string
	HelmReleaseName  string
	Version          string
	RecipeVersion    string

	// Extensions holds component-specific fields.
	// Templates can access these via {{ .Script.Extensions.FieldName }}
	Extensions map[string]interface{}
}

// GenerateDefaultBundleMetadata creates default bundle metadata from config map.
func GenerateDefaultBundleMetadata(config map[string]string, name string, defaultHelmRepo string, defaultHelmChart string) *BundleMetadata {
	return &BundleMetadata{
		Namespace:        GetConfigValue(config, "namespace", name),
		HelmRepository:   GetConfigValue(config, "helm_repository", defaultHelmRepo),
		HelmChart:        defaultHelmChart,
		HelmChartVersion: GetConfigValue(config, "helm_chart_version", ""),
		HelmReleaseName:  name,
		Version:          GetBundlerVersion(config),
		RecipeVersion:    GetRecipeBundlerVersion(config),
		Extensions:       make(map[string]interface{}),
	}
}

// GenerateBundleMetadataWithExtensions creates bundle metadata with custom extensions.
// This is used when components need additional fields beyond the standard ones.
func GenerateBundleMetadataWithExtensions(config map[string]string, cfg ComponentConfig) *BundleMetadata {
	meta := GenerateDefaultBundleMetadata(config, cfg.Name, cfg.DefaultHelmRepository, cfg.DefaultHelmChart)

	// Apply default chart version if specified in config
	if cfg.DefaultHelmChartVersion != "" && meta.HelmChartVersion == "" {
		meta.HelmChartVersion = cfg.DefaultHelmChartVersion
	}

	// Merge extensions from component config
	if cfg.MetadataExtensions != nil {
		for k, v := range cfg.MetadataExtensions {
			meta.Extensions[k] = v
		}
	}

	return meta
}

// MakeBundle generates a bundle using the generic bundling logic.
// This function handles the common steps: creating directories, applying overrides,
// writing values.yaml, generating README, generating checksums, and finalizing.
func MakeBundle(ctx context.Context, b *BaseBundler, input recipe.RecipeInput, outputDir string, cfg ComponentConfig) (*result.Result, error) {
	if err := ctx.Err(); err != nil {
		return nil, errors.Wrap(errors.ErrCodeTimeout, "context cancelled", err)
	}

	start := time.Now()

	slog.Debug("generating bundle",
		"component", cfg.Name,
		"output_dir", outputDir,
	)

	// Get component reference
	componentRef := input.GetComponentRef(cfg.Name)
	if componentRef == nil {
		return nil, errors.New(errors.ErrCodeInvalidRequest,
			cfg.Name+" component not found in recipe")
	}

	// Get values from component reference
	values, err := input.GetValuesForComponent(cfg.Name)
	if err != nil {
		return nil, errors.Wrap(errors.ErrCodeInternal,
			"failed to get values for "+cfg.Name, err)
	}

	// Apply user value overrides from --set flags
	if overrides := getValueOverridesForComponent(b, cfg); len(overrides) > 0 {
		if applyErr := ApplyMapOverrides(values, overrides); applyErr != nil {
			slog.Warn("failed to apply some value overrides to values map", "error", applyErr)
		}
	}

	// Apply system node selectors
	if selectors := b.Config.SystemNodeSelector(); len(selectors) > 0 {
		ApplyNodeSelectorOverrides(values, selectors, cfg.SystemNodeSelectorPaths...)
	}

	// Apply system tolerations
	if tolerations := b.Config.SystemNodeTolerations(); len(tolerations) > 0 {
		ApplyTolerationsOverrides(values, tolerations, cfg.SystemTolerationPaths...)
	}

	// Apply accelerated node selectors
	if selectors := b.Config.AcceleratedNodeSelector(); len(selectors) > 0 {
		ApplyNodeSelectorOverrides(values, selectors, cfg.AcceleratedNodeSelectorPaths...)
	}

	// Apply accelerated tolerations
	if tolerations := b.Config.AcceleratedNodeTolerations(); len(tolerations) > 0 {
		ApplyTolerationsOverrides(values, tolerations, cfg.AcceleratedTolerationPaths...)
	}

	// Create bundle directory structure
	dirs, err := b.CreateBundleDir(outputDir, cfg.Name)
	if err != nil {
		return b.Result, errors.Wrap(errors.ErrCodeInternal,
			"failed to create bundle directory", err)
	}

	// Build config map with base settings for metadata extraction
	configMap := b.BuildConfigMapFromInput(input)
	configMap["namespace"] = cfg.Name
	configMap["helm_repository"] = componentRef.Source
	configMap["helm_chart_version"] = componentRef.Version

	// Add accelerator from criteria if available (for custom manifest generation)
	if criteria := input.GetCriteria(); criteria != nil {
		configMap["accelerator"] = string(criteria.Accelerator)
	}

	// Serialize values to YAML with header
	header := ValuesHeader{
		ComponentName:  cfg.DisplayName,
		BundlerVersion: configMap["bundler_version"],
		RecipeVersion:  configMap["recipe_version"],
	}
	valuesYAML, err := MarshalYAMLWithHeader(values, header)
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

	// Generate custom manifests if the component has a CustomManifestFunc
	if cfg.CustomManifestFunc != nil {
		if _, err := cfg.CustomManifestFunc(ctx, b, values, configMap, dirs.Root); err != nil {
			return b.Result, err
		}
	}

	// Generate metadata for templates
	var metadata interface{}
	if cfg.MetadataFunc != nil {
		metadata = cfg.MetadataFunc(configMap)
	} else {
		metadata = GenerateBundleMetadataWithExtensions(configMap, cfg)
	}

	// Create combined data for README (values map + metadata)
	readmeData := map[string]interface{}{
		"Values": values,
		"Script": metadata, // "Script" key preserved for template compatibility
	}

	// Generate README (only if template exists)
	if b.Config.IncludeReadme() && cfg.TemplateGetter != nil {
		if _, hasReadme := cfg.TemplateGetter("README.md"); hasReadme {
			readmePath := filepath.Join(dirs.Root, "README.md")
			if err := b.GenerateFileFromTemplate(ctx, cfg.TemplateGetter, "README.md",
				readmePath, readmeData, 0644); err != nil {
				return b.Result, err
			}
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

	slog.Debug("bundle generated",
		"component", cfg.Name,
		"files", len(b.Result.Files),
		"size_bytes", b.Result.Size,
		"duration", b.Result.Duration.Round(time.Millisecond),
	)

	return b.Result, nil
}

// getValueOverridesForComponent retrieves value overrides for a component from config.
// It checks the component name first, then any alternative keys specified in the config.
func getValueOverridesForComponent(b *BaseBundler, cfg ComponentConfig) map[string]string {
	allOverrides := b.Config.ValueOverrides()
	if allOverrides == nil {
		return nil
	}

	// Check the component name first
	if overrides, ok := allOverrides[cfg.Name]; ok {
		return overrides
	}

	// Check alternative keys
	for _, key := range cfg.ValueOverrideKeys {
		if overrides, ok := allOverrides[key]; ok {
			return overrides
		}
	}

	return nil
}
