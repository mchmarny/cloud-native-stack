// Package component provides reusable component implementations for generating deployment bundles.
//
// This package contains concrete implementations of components (GPU Operator, Network Operator, etc.)
// that generate deployment artifacts from recipes. Each component is responsible for:
//
//   - Extracting configuration from recipes via component references
//   - Generating Helm values files with proper YAML headers
//   - Creating Kubernetes manifests (when needed)
//   - Computing checksums for verification
//
// Components self-register with the bundler registry via init() functions, enabling automatic
// discovery and execution without manual registration.
//
// # Architecture
//
// The component package uses a generic bundler framework based on [internal.ComponentConfig]
// and [internal.MakeBundle]. This eliminates boilerplate by:
//
//   - Centralizing common bundling logic in the internal package
//   - Using declarative configuration (ComponentConfig) instead of imperative code
//   - Providing default implementations with customization hooks
//
// Package structure:
//
//   - internal/: Generic bundler framework (ComponentConfig, MakeBundle, BaseBundler)
//   - certmanager/: Cert-Manager component with custom InstallCRDs metadata
//   - dradriver/: DRA Driver component for Dynamic Resource Allocation
//   - gpuoperator/: GPU Operator component with custom manifest generation
//   - networkoperator/: Network Operator component for RDMA and SR-IOV
//   - nvsentinel/: NVIDIA Sentinel component with custom metadata
//   - skyhook/: Skyhook Operator component with custom metadata
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
//  2. Define a [internal.ComponentConfig] with component-specific settings
//  3. Create a Bundler struct that embeds [internal.BaseBundler]
//  4. Implement Make() by calling [internal.MakeBundle] with your config
//  5. Self-register in init() using [registry.MustRegister]
//
// For components needing custom Kubernetes manifests, provide a CustomManifestFunc
// in the ComponentConfig with templates in a templates/ subdirectory.
//
// # Package Structure
//
// Each component package should contain:
//   - bundler.go: ComponentConfig, Bundler struct, NewBundler, Make
//   - bundler_test.go: Tests using [internal.RunStandardBundlerTests]
//   - doc.go: Package documentation
//   - templates/ (optional): Only needed for CustomManifestFunc templates
//
// See pkg/component/networkoperator for a minimal example, or
// pkg/component/gpuoperator for an example with custom manifest generation.
package component
