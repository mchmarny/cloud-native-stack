package bundler

import (
	"context"
	"fmt"

	"github.com/NVIDIA/cloud-native-stack/pkg/recipe"
)

// Bundler defines the interface for creating application bundles.
// Implementations generate deployment artifacts from recipes.
type Bundler interface {
	// Make generates the bundle in the specified directory.
	// Returns a BundleResult containing information about generated files.
	Make(ctx context.Context, recipe *recipe.Recipe, dir string) (*BundleResult, error)
}

// ConfigurableBundler extends Bundler with configuration support.
type ConfigurableBundler interface {
	Bundler
	// Configure applies configuration to the bundler.
	Configure(config *BundlerConfig) error
}

// BundleType identifies different types of bundles.
type BundleType string

const (
	// BundleTypeGpuOperator generates GPU Operator bundles.
	BundleTypeGpuOperator BundleType = "gpu-operator"
)

// ParseBundleType converts a string to a BundleType.
func ParseBundleType(s string) (BundleType, error) {
	switch s {
	case string(BundleTypeGpuOperator):
		return BundleTypeGpuOperator, nil
	default:
		return "", fmt.Errorf("unknown bundle type: %s", s)
	}
}

// SupportedBundleTypes returns a list of all supported bundle types.
func SupportedBundleTypes() []BundleType {
	return []BundleType{
		BundleTypeGpuOperator,
	}
}

// SupportedBundleTypesAsStrings returns supported bundle types as strings.
func SupportedBundleTypesAsStrings() []string {
	types := SupportedBundleTypes()
	strs := make([]string, len(types))
	for i, t := range types {
		strs[i] = string(t)
	}
	return strs
}
