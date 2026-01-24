package gpuoperator

import (
	"context"
	"log/slog"
	"path/filepath"
	"strings"
	"time"

	"github.com/NVIDIA/cloud-native-stack/pkg/bundler/config"
	"github.com/NVIDIA/cloud-native-stack/pkg/bundler/result"
	"github.com/NVIDIA/cloud-native-stack/pkg/bundler/types"
	common "github.com/NVIDIA/cloud-native-stack/pkg/component/internal"
	"github.com/NVIDIA/cloud-native-stack/pkg/errors"
	"github.com/NVIDIA/cloud-native-stack/pkg/recipe"
)

const (
	Name = "gpu-operator"
)

// Bundler creates GPU Operator application bundles based on recipes.
type Bundler struct {
	*common.BaseBundler
}

// NewBundler creates a new GPU Operator bundler instance.
func NewBundler(conf *config.Config) *Bundler {
	return &Bundler{
		BaseBundler: common.NewBaseBundler(conf, types.BundleTypeGpuOperator),
	}
}

// Make generates the GPU Operator bundle based on the provided recipe.
// Expects RecipeResult with component references and values maps.
func (b *Bundler) Make(ctx context.Context, input recipe.RecipeInput, dir string) (*result.Result, error) {
	start := time.Now()

	slog.Debug("generating GPU Operator bundle",
		"output_dir", dir,
		"namespace", Name,
	)

	// Get component reference for gpu-operator
	componentRef := input.GetComponentRef(Name)
	if componentRef == nil {
		return nil, errors.New(errors.ErrCodeInvalidRequest,
			Name+" component not found in recipe")
	}

	// Get values from component reference
	values, err := input.GetValuesForComponent(Name)
	if err != nil {
		return nil, errors.Wrap(errors.ErrCodeInternal,
			"failed to get values for gpu-operator", err)
	}

	// Apply user value overrides from --set flags to values map
	if overrides := b.getValueOverrides(); len(overrides) > 0 {
		if applyErr := common.ApplyMapOverrides(values, overrides); applyErr != nil {
			slog.Warn("failed to apply some value overrides to values map", "error", applyErr)
		}
	}

	// Apply system node selector (for operator control plane components)
	if nodeSelector := b.Config.SystemNodeSelector(); len(nodeSelector) > 0 {
		common.ApplyNodeSelectorOverrides(values, nodeSelector,
			"operator.nodeSelector",
			"node-feature-discovery.gc.nodeSelector",
			"node-feature-discovery.master.nodeSelector",
		)
	}

	// Apply system node tolerations (for operator control plane components)
	if tolerations := b.Config.SystemNodeTolerations(); len(tolerations) > 0 {
		common.ApplyTolerationsOverrides(values, tolerations,
			"operator.tolerations",
			"node-feature-discovery.gc.tolerations",
			"node-feature-discovery.master.tolerations",
		)
	}

	// Apply accelerated node selector (for GPU node daemonsets)
	if nodeSelector := b.Config.AcceleratedNodeSelector(); len(nodeSelector) > 0 {
		common.ApplyNodeSelectorOverrides(values, nodeSelector,
			"daemonsets.nodeSelector",
			"node-feature-discovery.worker.nodeSelector",
		)
	}

	// Apply accelerated node tolerations (for GPU node daemonsets)
	if tolerations := b.Config.AcceleratedNodeTolerations(); len(tolerations) > 0 {
		common.ApplyTolerationsOverrides(values, tolerations,
			"daemonsets.tolerations",
			"node-feature-discovery.worker.tolerations",
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
		ComponentName:  "GPU Operator",
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

	// Get criteria to make accelerator-specific decisions
	criteria := input.GetCriteria()

	// Generate manifests using values map directly
	if err := b.generateManifestsFromData(ctx, values, metadata, criteria, dirs.Root); err != nil {
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

	slog.Debug("GPU Operator bundle generated from recipe result",
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

// generateManifestsFromData generates manifests from pre-built data.
func (b *Bundler) generateManifestsFromData(ctx context.Context, values map[string]interface{}, metadata *BundleMetadata, criteria *recipe.Criteria, dir string) error {
	// Combine values map with bundle metadata for template
	manifestData := map[string]interface{}{
		"Values": values,
		"Script": metadata, // "Script" key preserved for template compatibility
	}

	// Generate DCGM Exporter ConfigMap if enabled
	if shouldGenerateDCGMExporterConfigMap(values) {
		dcgmPath := filepath.Join(dir, "manifests", "dcgm-exporter.yaml")
		if err := b.GenerateFileFromTemplate(ctx, GetTemplate, "dcgm-exporter",
			dcgmPath, manifestData, 0644); err != nil {
			return err
		}
	}

	// Generate Kernel Module Params ConfigMap for GB200 accelerator
	if shouldGenerateKernelModuleConfigMap(criteria) {
		kmpPath := filepath.Join(dir, "manifests", "kernel-module-params.yaml")
		if err := b.GenerateFileFromTemplate(ctx, GetTemplate, "kernel-module-params",
			kmpPath, manifestData, 0644); err != nil {
			return err
		}
	}

	return nil
}

// shouldGenerateDCGMExporterConfigMap checks if DCGM exporter ConfigMap should be generated.
// Returns true if dcgmExporter.config.create is true.
func shouldGenerateDCGMExporterConfigMap(values map[string]interface{}) bool {
	dcgmExporter, ok := values["dcgmExporter"].(map[string]interface{})
	if !ok {
		return false
	}
	config, ok := dcgmExporter["config"].(map[string]interface{})
	if !ok {
		return false
	}
	create, ok := config["create"].(bool)
	return ok && create
}

// shouldGenerateKernelModuleConfigMap checks if kernel module params ConfigMap should be generated.
// Returns true for GB200 accelerator which requires special kernel module parameters.
func shouldGenerateKernelModuleConfigMap(criteria *recipe.Criteria) bool {
	if criteria == nil {
		return false
	}
	// GB200 requires kernel module params ConfigMap for NVreg settings
	return strings.EqualFold(string(criteria.Accelerator), "gb200")
}

// getValueOverrides retrieves value overrides for this bundler from config.
func (b *Bundler) getValueOverrides() map[string]string {
	allOverrides := b.Config.ValueOverrides()
	if allOverrides == nil {
		return nil
	}
	// Return overrides for "gpuoperator" or "gpu-operator"
	if overrides, ok := allOverrides["gpuoperator"]; ok {
		return overrides
	}
	if overrides, ok := allOverrides["gpu-operator"]; ok {
		return overrides
	}
	return nil
}
