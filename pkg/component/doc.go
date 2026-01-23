// Package component provides reusable component implementations for generating deployment bundles.
//
// This package contains concrete implementations of components (GPU Operator, Network Operator, etc.)
// that generate deployment artifacts from recipes. Each component is responsible for:
//
//   - Extracting configuration from recipes
//   - Generating Helm values files
//   - Creating Kubernetes manifests
//   - Producing installation/uninstallation scripts
//   - Computing checksums for verification
//
// Components self-register with the bundler registry via init() functions, enabling automatic
// discovery and execution without manual registration.
//
// # Architecture
//
// The component package is structured as:
//
//   - internal/: Base component implementation (BaseBundler) and shared utilities
//   - certmanager/: Cert-Manager component for certificate management
//   - gpuoperator/: GPU Operator component for GPU workload management
//   - networkoperator/: Network Operator component for RDMA and SR-IOV
//   - nvsentinel/: NVIDIA Sentinel component for monitoring
//   - skyhook/: Skyhook Operator component for node optimization (skyhook-operator)
//
// # Usage
//
// Components are automatically registered and executed by the bundler orchestrator:
//
//	import (
//	    _ "github.com/NVIDIA/cloud-native-stack/pkg/component/gpuoperator"  // Auto-registers
//	)
//
// The bundler package in pkg/bundler handles orchestration, while this package provides
// the component implementations.
//
// # Adding New Components
//
// To add a new component:
//
//  1. Create a new package under pkg/component/<name>
//  2. Implement the Bundler interface from pkg/bundler/types
//  3. Embed BaseBundler from pkg/component/internal for common functionality
//  4. Self-register in init() using bundler.MustRegister()
//  5. Add templates using go:embed directive
//
// See pkg/component/gpuoperator for a complete reference implementation.
package component
