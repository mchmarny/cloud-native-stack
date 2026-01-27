// Package types defines the type system for bundler implementations.
//
// This package provides a type-safe way to identify and work with different
// bundler types throughout the framework.
//
// # Core Type
//
// BundleType: String-based type identifier for bundlers
//
//	type BundleType string
//
// # Component Names
//
// Component names are defined declaratively in pkg/recipe/data/registry.yaml.
// BundleType values are created from these names at runtime:
//
//	bundlerType := types.BundleType("gpu-operator")
//	fmt.Println(bundlerType.String()) // Output: gpu-operator
//
// # Map Keys
//
// BundleType can be used as map keys:
//
//	bundlers := map[types.BundleType]Bundler{
//	    types.BundleType("gpu-operator"):     gpuBundler,
//	    types.BundleType("network-operator"): networkBundler,
//	}
//
// # Type Comparison
//
// Types can be compared directly:
//
//	if bundlerType == types.BundleType("gpu-operator") {
//	    // Handle GPU Operator
//	}
//
// # Adding New Components
//
// To add a new component, add an entry to pkg/recipe/data/registry.yaml.
// No Go code changes are required.
//
// # Zero Value
//
// The zero value of BundleType is an empty string "".
package types
