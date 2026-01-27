package types

// BundleType represents the type of a bundler.
// Component names are defined in pkg/recipe/data/registry.yaml.
type BundleType string

// String returns the string representation of the bundle type.
func (bt BundleType) String() string {
	return string(bt)
}
