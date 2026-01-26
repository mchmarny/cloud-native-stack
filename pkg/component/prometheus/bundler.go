package prometheus

import (
	"context"

	"github.com/NVIDIA/cloud-native-stack/pkg/bundler/config"
	"github.com/NVIDIA/cloud-native-stack/pkg/bundler/registry"
	"github.com/NVIDIA/cloud-native-stack/pkg/bundler/result"
	"github.com/NVIDIA/cloud-native-stack/pkg/bundler/types"
	"github.com/NVIDIA/cloud-native-stack/pkg/component/internal"
	"github.com/NVIDIA/cloud-native-stack/pkg/recipe"
)

const (
	Name = "prometheus"
)

func init() {
	// Register Prometheus bundler factory in global registry
	registry.MustRegister(types.BundleTypePrometheus, func(cfg *config.Config) registry.Bundler {
		return NewBundler(cfg)
	})
}

// componentConfig defines the Prometheus bundler configuration.
var componentConfig = internal.ComponentConfig{
	Name:                    Name,
	DisplayName:             "prometheus",
	ValueOverrideKeys:       []string{"prometheus"},
	DefaultHelmRepository:   "https://prometheus-community.github.io/helm-charts",
	DefaultHelmChart:        "prometheus-community/prometheus",
	DefaultHelmChartVersion: "25.29.0",
	SystemNodeSelectorPaths: []string{
		"server.nodeSelector",
		"alertmanager.nodeSelector",
		"pushgateway.nodeSelector",
	},
	SystemTolerationPaths: []string{
		"server.tolerations",
		"alertmanager.tolerations",
		"pushgateway.tolerations",
	},
	AcceleratedNodeSelectorPaths: []string{
		"nodeExporter.nodeSelector",
	},
	AcceleratedTolerationPaths: []string{
		"nodeExporter.tolerations",
	},
}

// Bundler generates Prometheus deployment bundles.
type Bundler struct {
	*internal.BaseBundler
}

// NewBundler creates a new Prometheus bundler.
func NewBundler(cfg *config.Config) *Bundler {
	return &Bundler{
		BaseBundler: internal.NewBaseBundler(cfg, types.BundleTypePrometheus),
	}
}

// Make generates a Prometheus bundle from a recipe.
func (b *Bundler) Make(ctx context.Context, input recipe.RecipeInput, outputDir string) (*result.Result, error) {
	return internal.MakeBundle(ctx, b.BaseBundler, input, outputDir, componentConfig)
}