package dradriver

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
	Name = "nvidia-dra-driver-gpu"
)

func init() {
	// Register DRA Driver bundler factory in global registry
	registry.MustRegister(types.BundleTypeDraDriver, func(cfg *config.Config) registry.Bundler {
		return NewBundler(cfg)
	})
}

// componentConfig defines the NVIDIA DRA Driver bundler configuration.
var componentConfig = common.ComponentConfig{
	Name:              Name,
	DisplayName:       "nvidia-dra-driver-gpu",
	ValueOverrideKeys: []string{"dradriver"},
	SystemTolerationPaths: []string{
		"controller.tolerations",
	},
	AcceleratedTolerationPaths: []string{
		"kubeletPlugin.tolerations",
	},
	DefaultHelmRepository: "https://helm.ngc.nvidia.com/nvidia",
	DefaultHelmChart:      "nvidia/nvidia-dra-driver-gpu",
}

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
func (b *Bundler) Make(ctx context.Context, input recipe.RecipeInput, dir string) (*result.Result, error) {
	return common.MakeBundle(ctx, b.BaseBundler, input, dir, componentConfig)
}
