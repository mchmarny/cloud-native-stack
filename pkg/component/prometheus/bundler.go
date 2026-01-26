package prometheus

import (
	"context"

	"github.com/NVIDIA/cloud-native-stack/pkg/bundler/config"
	"github.com/NVIDIA/cloud-native-stack/pkg/bundler/registry"
	"github.com/NVIDIA/cloud-native-stack/pkg/bundler/result"
	"github.com/NVIDIA/cloud-native-stack/pkg/bundler/types"
	common "github.com/NVIDIA/cloud-native-stack/pkg/component/internal"
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
var componentConfig = common.ComponentConfig{
	Name:                  Name,
	DisplayName:           "prometheus",
	ValueOverrideKeys:     []string{"prometheus"},
	DefaultHelmRepository: "https://prometheus-community.github.io/helm-charts",
	DefaultHelmChart:      "prometheus-community/prometheus",
	SystemTolerationPaths: []string{
		"server.tolerations",
		"alertmanager.tolerations",
	},
	AcceleratedTolerationPaths: []string{
		"nodeExporter.tolerations",
	},
}

// Bundler creates Prometheus bundles based on recipes.
type Bundler struct {
	*common.BaseBundler
}

// NewBundler creates a new Prometheus bundler instance.
func NewBundler(conf *config.Config) *Bundler {
	return &Bundler{
		BaseBundler: common.NewBaseBundler(conf, types.BundleTypePrometheus),
	}
}

// Make generates the Prometheus bundle based on the provided recipe.
func (b *Bundler) Make(ctx context.Context, input recipe.RecipeInput, dir string) (*result.Result, error) {
	return common.MakeBundle(ctx, b.BaseBundler, input, dir, componentConfig)
}