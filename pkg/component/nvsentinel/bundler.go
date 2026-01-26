package nvsentinel

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
	Name = "nvsentinel"
)

func init() {
	// Register NVSentinel bundler factory in global registry
	registry.MustRegister(types.BundleTypeNVSentinel, func(cfg *config.Config) registry.Bundler {
		return NewBundler(cfg)
	})
}

// componentConfig defines the NVSentinel bundler configuration.
var componentConfig = common.ComponentConfig{
	Name:                    Name,
	DisplayName:             "nvsentinel",
	ValueOverrideKeys:       []string{"nv-sentinel"},
	DefaultHelmRepository:   "https://helm.ngc.nvidia.com/nvidia",
	DefaultHelmChart:        "nvidia/nvsentinel",
	DefaultHelmChartVersion: "v0.6.0",
	SystemNodeSelectorPaths: []string{
		"global.systemNodeSelector",
	},
	AcceleratedTolerationPaths: []string{
		"global.tolerations",
	},
}

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
	return common.MakeBundle(ctx, b.BaseBundler, input, dir, componentConfig)
}
