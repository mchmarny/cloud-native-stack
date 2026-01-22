package dradriver

import (
	"github.com/NVIDIA/cloud-native-stack/pkg/bundler/config"
	"github.com/NVIDIA/cloud-native-stack/pkg/bundler/registry"
	"github.com/NVIDIA/cloud-native-stack/pkg/bundler/types"
)

func init() {
	// Register DRA Driver bundler factory in global registry
	registry.MustRegister(types.BundleTypeDraDriver, func(cfg *config.Config) registry.Bundler {
		return NewBundler(cfg)
	})
}
