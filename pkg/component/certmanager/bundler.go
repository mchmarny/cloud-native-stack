package certmanager

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
	Name = "cert-manager"
)

func init() {
	// Register cert-manager bundler factory in global registry
	registry.MustRegister(types.BundleTypeCertManager, func(cfg *config.Config) registry.Bundler {
		return NewBundler(cfg)
	})
}

// componentConfig defines the cert-manager bundler configuration.
var componentConfig = internal.ComponentConfig{
	Name:                    Name,
	DisplayName:             "cert-manager",
	ValueOverrideKeys:       []string{"certmanager"},
	DefaultHelmRepository:   "https://charts.jetstack.io",
	DefaultHelmChart:        "jetstack/cert-manager",
	DefaultHelmChartVersion: "v1.17.2",
	SystemNodeSelectorPaths: []string{
		"nodeSelector",
		"webhook.nodeSelector",
		"cainjector.nodeSelector",
		"startupapicheck.nodeSelector",
	},
	SystemTolerationPaths: []string{
		"tolerations",
		"webhook.tolerations",
		"cainjector.tolerations",
		"startupapicheck.tolerations",
	},
}

// Bundler generates cert-manager deployment bundles.
type Bundler struct {
	*internal.BaseBundler
}

// NewBundler creates a new cert-manager bundler.
func NewBundler(cfg *config.Config) *Bundler {
	return &Bundler{
		BaseBundler: internal.NewBaseBundler(cfg, types.BundleTypeCertManager),
	}
}

// Make generates a cert-manager bundle from a recipe.
func (b *Bundler) Make(ctx context.Context, input recipe.RecipeInput, outputDir string) (*result.Result, error) {
	return internal.MakeBundle(ctx, b.BaseBundler, input, outputDir, componentConfig)
}
