package common

import (
	"context"
	"fmt"

	"github.com/NVIDIA/cloud-native-stack/pkg/recipe"
)

const (
	// BundleTypeGpuOperator generates GPU Operator bundles.
	BundleTypeGpuOperator BundleType = "gpu-operator"

	// BundleTypeNetworkOperator generates Network Operator bundles.
	BundleTypeNetworkOperator BundleType = "network-operator"
)

// BundleType identifies different types of bundles.
type BundleType string

// Bundler defines the interface for creating application bundles.
// Implementations generate deployment artifacts from recipes.
type Bundler interface {
	Make(ctx context.Context, recipe *recipe.Recipe, dir string) (*Result, error)
}

// ParseType converts a string to a BundleType.
func ParseType(s string) (BundleType, error) {
	switch s {
	case string(BundleTypeGpuOperator):
		return BundleTypeGpuOperator, nil
	case string(BundleTypeNetworkOperator):
		return BundleTypeNetworkOperator, nil
	default:
		return "", fmt.Errorf("unknown bundle type: %s", s)
	}
}

// SupportedTypes returns a list of all supported bundle types.
func SupportedTypes() []BundleType {
	return []BundleType{
		BundleTypeGpuOperator,
		BundleTypeNetworkOperator,
	}
}

// SupportedBundleTypesAsStrings returns supported bundle types as strings.
func SupportedBundleTypesAsStrings() []string {
	types := SupportedTypes()
	strs := make([]string, len(types))
	for i, t := range types {
		strs[i] = string(t)
	}
	return strs
}
