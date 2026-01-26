package gpuoperator

import (
	"context"
	_ "embed"
	"path/filepath"
	"strings"

	"github.com/NVIDIA/cloud-native-stack/pkg/bundler/config"
	"github.com/NVIDIA/cloud-native-stack/pkg/bundler/registry"
	"github.com/NVIDIA/cloud-native-stack/pkg/bundler/result"
	"github.com/NVIDIA/cloud-native-stack/pkg/bundler/types"
	common "github.com/NVIDIA/cloud-native-stack/pkg/component/internal"
	"github.com/NVIDIA/cloud-native-stack/pkg/recipe"
)

const (
	Name                  = "gpu-operator"
	DefaultHelmRepository = "https://helm.ngc.nvidia.com/nvidia"
	DefaultHelmChart      = "nvidia/gpu-operator"
)

var (
	//go:embed templates/kernel-module-params.yaml.tmpl
	kernelModuleParamsTemplate string

	//go:embed templates/dcgm-exporter.yaml.tmpl
	dcgmExporterTemplate string

	// GetTemplate returns the named template content for manifest generation.
	GetTemplate = common.NewTemplateGetter(map[string]string{
		"kernel-module-params": kernelModuleParamsTemplate,
		"dcgm-exporter":        dcgmExporterTemplate,
	})
)

func init() {
	// Register GPU Operator bundler factory in global registry
	registry.MustRegister(types.BundleTypeGpuOperator, func(cfg *config.Config) registry.Bundler {
		return NewBundler(cfg)
	})
}

// componentConfig defines the GPU Operator bundler configuration.
var componentConfig = common.ComponentConfig{
	Name:              Name,
	DisplayName:       "gpu-operator",
	ValueOverrideKeys: []string{"gpuoperator"},
	SystemNodeSelectorPaths: []string{
		"operator.nodeSelector",
		"node-feature-discovery.gc.nodeSelector",
		"node-feature-discovery.master.nodeSelector",
	},
	SystemTolerationPaths: []string{
		"operator.tolerations",
		"node-feature-discovery.gc.tolerations",
		"node-feature-discovery.master.tolerations",
	},
	AcceleratedNodeSelectorPaths: []string{
		"daemonsets.nodeSelector",
		"node-feature-discovery.worker.nodeSelector",
	},
	AcceleratedTolerationPaths: []string{
		"daemonsets.tolerations",
		"node-feature-discovery.worker.tolerations",
	},
	DefaultHelmRepository: DefaultHelmRepository,
	DefaultHelmChart:      DefaultHelmChart,
	TemplateGetter:        GetTemplate,
	CustomManifestFunc:    generateCustomManifests,
}

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
func (b *Bundler) Make(ctx context.Context, input recipe.RecipeInput, dir string) (*result.Result, error) {
	return common.MakeBundle(ctx, b.BaseBundler, input, dir, componentConfig)
}

// generateCustomManifests generates GPU Operator-specific manifests (DCGM exporter, kernel module params).
func generateCustomManifests(ctx context.Context, b *common.BaseBundler, values map[string]interface{}, configMap map[string]string, dir string) ([]string, error) {
	var generatedFiles []string

	// Generate bundle metadata for manifest templates using the internal default function
	metadata := common.GenerateDefaultBundleMetadata(configMap, Name, DefaultHelmRepository, DefaultHelmChart)
	manifestData := map[string]interface{}{
		"Values": values,
		"Script": metadata,
	}

	// Generate DCGM Exporter ConfigMap if enabled
	if shouldGenerateDCGMExporterConfigMap(values) {
		dcgmPath := filepath.Join(dir, "manifests", "dcgm-exporter.yaml")
		if err := b.GenerateFileFromTemplate(ctx, GetTemplate, "dcgm-exporter",
			dcgmPath, manifestData, 0644); err != nil {
			return generatedFiles, err
		}
		generatedFiles = append(generatedFiles, dcgmPath)
	}

	// Generate Kernel Module Params ConfigMap for GB200 accelerator
	// We need criteria from a different source - check configMap for accelerator
	if shouldGenerateKernelModuleConfigMapFromConfig(configMap) {
		kmpPath := filepath.Join(dir, "manifests", "kernel-module-params.yaml")
		if err := b.GenerateFileFromTemplate(ctx, GetTemplate, "kernel-module-params",
			kmpPath, manifestData, 0644); err != nil {
			return generatedFiles, err
		}
		generatedFiles = append(generatedFiles, kmpPath)
	}

	return generatedFiles, nil
}

// shouldGenerateDCGMExporterConfigMap checks if DCGM exporter ConfigMap should be generated.
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

// shouldGenerateKernelModuleConfigMapFromConfig checks if kernel module params ConfigMap should be generated.
func shouldGenerateKernelModuleConfigMapFromConfig(configMap map[string]string) bool {
	accelerator := configMap["accelerator"]
	return strings.EqualFold(accelerator, "gb200")
}
