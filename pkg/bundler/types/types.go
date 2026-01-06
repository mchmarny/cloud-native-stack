package types

import (
	"fmt"
)

// BundleType represents the type of a bundler.
type BundleType string

// Supported bundler types.
const (
	BundleTypeGpuOperator     BundleType = "gpu-operator"
	BundleTypeNetworkOperator BundleType = "network-operator"
	BundleTypeSkyhook         BundleType = "skyhook"
)

// String returns the string representation of the bundle type.
func (bt BundleType) String() string {
	return string(bt)
}

// ParseType converts a string to a BundleType.
// Returns an error if the string is not a valid bundle type.
func ParseType(s string) (BundleType, error) {
	switch s {
	case string(BundleTypeGpuOperator):
		return BundleTypeGpuOperator, nil
	case string(BundleTypeNetworkOperator):
		return BundleTypeNetworkOperator, nil
	case string(BundleTypeSkyhook):
		return BundleTypeSkyhook, nil
	default:
		return "", fmt.Errorf("unsupported bundle type: %s", s)
	}
}

// SupportedTypes returns all supported bundle types.
func SupportedTypes() []BundleType {
	return []BundleType{
		BundleTypeSkyhook,
		BundleTypeGpuOperator,
		BundleTypeNetworkOperator,
	}
}

// SupportedBundleTypesAsStrings returns all supported bundle types as strings.
func SupportedBundleTypesAsStrings() []string {
	types := SupportedTypes()
	result := make([]string, len(types))
	for i, t := range types {
		result[i] = string(t)
	}
	return result
}
