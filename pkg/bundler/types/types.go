package types

import (
	"fmt"
	"strings"

	"github.com/agnivade/levenshtein"
)

// BundleType represents the type of a bundler.
type BundleType string

// Supported bundler types.
const (
	BundleTypeGpuOperator     BundleType = "gpu-operator"
	BundleTypeNetworkOperator BundleType = "network-operator"
	BundleTypeSkyhook         BundleType = "skyhook"
	BundleTypeNVSentinel      BundleType = "nvsentinel"
	BundleTypeCertManager     BundleType = "cert-manager"
	BundleTypeDraDriver       BundleType = "dra-driver"
)

// String returns the string representation of the bundle type.
func (bt BundleType) String() string {
	return string(bt)
}

// ParseType converts a string to a BundleType.
// Matching is case-insensitive. Returns an error with a suggestion if the
// string is not a valid bundle type but is close to one.
func ParseType(s string) (BundleType, error) {
	lower := strings.ToLower(s)
	switch lower {
	case string(BundleTypeGpuOperator):
		return BundleTypeGpuOperator, nil
	case string(BundleTypeNetworkOperator):
		return BundleTypeNetworkOperator, nil
	case string(BundleTypeSkyhook):
		return BundleTypeSkyhook, nil
	case string(BundleTypeNVSentinel):
		return BundleTypeNVSentinel, nil
	case string(BundleTypeCertManager):
		return BundleTypeCertManager, nil
	case string(BundleTypeDraDriver):
		return BundleTypeDraDriver, nil
	default:
		if suggestion := findClosestBundleType(lower); suggestion != "" {
			return "", fmt.Errorf("unsupported bundle type %q (did you mean %q?)", s, suggestion)
		}
		return "", fmt.Errorf("unsupported bundle type %q", s)
	}
}

// findClosestBundleType finds the closest matching bundle type using Levenshtein distance.
// Returns empty string if no close match is found (distance > maxDistance).
func findClosestBundleType(input string) string {
	const maxDistance = 5 // Maximum edit distance to consider a suggestion

	var closest string
	minDist := maxDistance + 1

	for _, bt := range SupportedTypes() {
		dist := levenshtein.ComputeDistance(input, string(bt))
		if dist < minDist {
			minDist = dist
			closest = string(bt)
		}
	}

	if minDist <= maxDistance {
		return closest
	}
	return ""
}

// SupportedTypes returns all supported bundle types.
func SupportedTypes() []BundleType {
	return []BundleType{
		BundleTypeSkyhook,
		BundleTypeGpuOperator,
		BundleTypeNetworkOperator,
		BundleTypeNVSentinel,
		BundleTypeCertManager,
		BundleTypeDraDriver,
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
