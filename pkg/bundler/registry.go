package bundler

import (
	"fmt"
	"sync"

	"github.com/NVIDIA/cloud-native-stack/pkg/bundler/common"
	"github.com/NVIDIA/cloud-native-stack/pkg/bundler/config"
	"github.com/NVIDIA/cloud-native-stack/pkg/bundler/gpuoperator"
	"github.com/NVIDIA/cloud-native-stack/pkg/bundler/networkoperator"
)

// Registry manages registered bundlers with thread-safe operations.
type Registry struct {
	bundlers map[common.BundleType]common.Bundler

	mu sync.RWMutex
}

// NewRegistry creates a new Registry instance.
func NewRegistry(cfg *config.Config) *Registry {
	return &Registry{
		bundlers: map[common.BundleType]common.Bundler{
			common.BundleTypeGpuOperator:     gpuoperator.NewBundler(cfg),
			common.BundleTypeNetworkOperator: networkoperator.NewBundler(cfg),
		},
	}
}

// Register registers a bundler in this registry.
func (r *Registry) Register(bundleType common.BundleType, b common.Bundler) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.bundlers[bundleType] = b
}

// Get retrieves a bundler by type from this registry.
func (r *Registry) Get(bundleType common.BundleType) (common.Bundler, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	b, ok := r.bundlers[bundleType]
	return b, ok
}

// GetAll returns all registered bundlers.
func (r *Registry) GetAll() map[common.BundleType]common.Bundler {
	r.mu.RLock()
	defer r.mu.RUnlock()

	bundlers := make(map[common.BundleType]common.Bundler, len(r.bundlers))
	for k, v := range r.bundlers {
		bundlers[k] = v
	}
	return bundlers
}

// List returns all registered bundler types.
func (r *Registry) List() []common.BundleType {
	r.mu.RLock()
	defer r.mu.RUnlock()

	types := make([]common.BundleType, 0, len(r.bundlers))
	for k := range r.bundlers {
		types = append(types, k)
	}
	return types
}

// Unregister removes a bundler from this registry.
func (r *Registry) Unregister(bundleType common.BundleType) error {
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
