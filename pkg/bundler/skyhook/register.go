package skyhook

import (
	"github.com/NVIDIA/cloud-native-stack/pkg/bundler/config"
	"github.com/NVIDIA/cloud-native-stack/pkg/bundler/registry"
	"github.com/NVIDIA/cloud-native-stack/pkg/bundler/types"
)

func init() {
	// Register Skyhook bundler factory in global registry
	registry.MustRegister(types.BundleTypeSkyhook, func(cfg *config.Config) registry.Bundler {
		return NewBundler(cfg)
	})
}
