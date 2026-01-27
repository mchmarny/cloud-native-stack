// Package registry provides thread-safe registration and retrieval of bundler implementations.
//
// The registry enables a plugin-like architecture where bundler implementations can
// self-register during package initialization, and the framework can discover and
// instantiate them dynamically at runtime.
//
// # Core Types
//
// Registry: Thread-safe storage for bundler factory functions
//
//	type Registry struct {
//	    mu        sync.RWMutex
//	    factories map[types.BundleType]Factory
//	}
//
// Factory: Function that creates a bundler instance
//
//	type Factory func(cfg *config.Config) Bundler
//
// Bundler: Interface that all bundler implementations must satisfy
//
//	type Bundler interface {
//	    Type() types.BundleType
//	    Validate(ctx context.Context, r *recipe.Recipe) error
//	    Make(ctx context.Context, r *recipe.Recipe, outputDir string) (*result.Result, error)
//	}
//
// # Registration Pattern
//
// Component names are defined in pkg/recipe/data/registry.yaml.
// Bundlers self-register in their package init() functions:
//
//	package gpuoperator
//
//	import (
//	    "github.com/NVIDIA/cloud-native-stack/pkg/bundler/registry"
//	    "github.com/NVIDIA/cloud-native-stack/pkg/bundler/types"
//	)
//
//	func init() {
//	    registry.MustRegister(types.BundleType("gpu-operator"), func(cfg *config.Config) registry.Bundler {
//	        return NewBundler(cfg)
//	    })
//	}
//
// The MustRegister function panics on duplicate registration, ensuring early
// detection of configuration errors.
//
// # Usage - Global Registry
//
// Access the global registry instance:
//
//	reg := registry.Global()
//
// Get all registered types:
//
//	types := reg.Types()
//	fmt.Printf("Available bundlers: %v\n", types)
//
// Check if a type is registered:
//
//	if reg.Has(types.BundleType("gpu-operator")) {
//	    // GPU Operator bundler is available
//	}
//
// Get a bundler instance:
//
//	bundler, ok := reg.Get(types.BundleType("gpu-operator"))
//	if ok {
//	    result, err := bundler.Make(ctx, recipe, outputDir)
//	}
//
// Get multiple bundlers:
//
//	bundlers := reg.GetAll([]types.BundleType{
//	    types.BundleType("gpu-operator"),
//	    types.BundleType("network-operator"),
//	})
//
// # Usage - Custom Registry
//
// Create a custom registry for testing:
//
//	reg := registry.New()
//	reg.Register(types.BundleType("gpu-operator"), func(cfg *config.Config) registry.Bundler {
//	    return mockBundler
//	})
//
// # Thread Safety
//
// The registry uses sync.RWMutex for safe concurrent access:
//   - Reads (Get, Has, Types, GetAll) acquire read locks
//   - Writes (Register, MustRegister) acquire write locks
//
// This allows multiple bundlers to be retrieved concurrently during parallel
// bundle generation.
//
// # Error Handling
//
// Register returns an error if a type is already registered:
//
//	err := reg.Register(types.BundleType("gpu-operator"), factory)
//	if err != nil {
//	    // Handle duplicate registration
//	}
//
// MustRegister panics on duplicate registration:
//
//	reg.MustRegister(types.BundleType("gpu-operator"), factory)
//	// Panics if already registered
//
// # Discovery
//
// The framework uses the registry for dynamic discovery:
//
//	// Get all available bundler types
//	available := registry.Global().Types()
//
//	// Create instances for all types
//	config := config.NewConfig()
//	bundlers := registry.Global().GetAll(available)
//
//	// Execute bundlers in parallel
//	for _, b := range bundlers {
//	    go b.Make(ctx, recipe, outputDir)
//	}
//
// # Testing
//
// Create isolated registries for testing:
//
//	func TestMyBundler(t *testing.T) {
//	    reg := registry.New()
//	    reg.Register(types.BundleType("gpu-operator"), func(cfg *config.Config) registry.Bundler {
//	        return &mockBundler{}
//	    })
//
//	    bundler, ok := reg.Get(types.BundleType("gpu-operator"))
//	    if !ok {
//	        t.Fatal("bundler not found")
//	    }
//	    // Test bundler...
//	}
package registry
