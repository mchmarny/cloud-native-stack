package bundler

import (
	"fmt"
	"sync"

	"github.com/NVIDIA/cloud-native-stack/pkg/bundler/bundle"
	"github.com/NVIDIA/cloud-native-stack/pkg/bundler/config"
	"github.com/NVIDIA/cloud-native-stack/pkg/bundler/gpuoperator"
)

// Registry manages registered bundlers with thread-safe operations.
type Registry struct {
	bundlers map[bundle.Type]bundle.Bundler

	mu sync.RWMutex
}

// NewRegistry creates a new Registry instance.
func NewRegistry(cfg *config.Config) *Registry {
	return &Registry{
		bundlers: map[bundle.Type]bundle.Bundler{
			bundle.BundleTypeGpuOperator: gpuoperator.NewBundler(cfg),
		},
	}
}

// Register registers a bundler in this registry.
func (r *Registry) Register(bundleType bundle.Type, b bundle.Bundler) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.bundlers[bundleType] = b
}

// Get retrieves a bundler by type from this registry.
func (r *Registry) Get(bundleType bundle.Type) (bundle.Bundler, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	b, ok := r.bundlers[bundleType]
	return b, ok
}

// GetAll returns all registered bundlers.
func (r *Registry) GetAll() map[bundle.Type]bundle.Bundler {
	r.mu.RLock()
	defer r.mu.RUnlock()

	bundlers := make(map[bundle.Type]bundle.Bundler, len(r.bundlers))
	for k, v := range r.bundlers {
		bundlers[k] = v
	}
	return bundlers
}

// List returns all registered bundler types.
func (r *Registry) List() []bundle.Type {
	r.mu.RLock()
	defer r.mu.RUnlock()

	types := make([]bundle.Type, 0, len(r.bundlers))
	for k := range r.bundlers {
		types = append(types, k)
	}
	return types
}

// Unregister removes a bundler from this registry.
func (r *Registry) Unregister(bundleType bundle.Type) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.bundlers[bundleType]; !ok {
		return fmt.Errorf("bundler type %s not registered", bundleType)
	}

	delete(r.bundlers, bundleType)
	return nil
}

// Count returns the number of registered bundlers.
func (r *Registry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.bundlers)
}

// IsEmpty returns true if no bundlers are registered.
// This is useful for checking if a registry has been populated.
func (r *Registry) IsEmpty() bool {
	return r.Count() == 0
}
