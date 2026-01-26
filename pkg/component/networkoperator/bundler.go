package networkoperator

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
	Name = "network-operator"
)

func init() {
	// Register Network Operator bundler factory in global registry
	registry.MustRegister(types.BundleTypeNetworkOperator, func(cfg *config.Config) registry.Bundler {
		return NewBundler(cfg)
	})
}

// componentConfig defines the Network Operator bundler configuration.
var componentConfig = internal.ComponentConfig{
	Name:                  Name,
	DisplayName:           "network-operator",
	ValueOverrideKeys:     []string{"networkoperator"},
	DefaultHelmRepository: "https://helm.ngc.nvidia.com/nvidia",
	DefaultHelmChart:      "nvidia/network-operator",
}

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
	return internal.MakeBundle(ctx, b.BaseBundler, input, outputDir, componentConfig)
}
