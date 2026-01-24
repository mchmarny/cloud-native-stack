// Package skyhook implements bundle generation for Skyhook node optimization.
//
// Skyhook provides automated node-level optimizations for GPU-accelerated
// Kubernetes clusters, enabling:
//   - Kernel parameter tuning for optimal GPU performance
//   - NUMA-aware scheduling configuration
//   - Memory management optimization (huge pages, transparent huge pages)
//   - Network stack tuning for RDMA and high-performance networking
//   - Power management settings for consistent GPU performance
//
// # Bundle Structure
//
// Generated bundles include:
//   - values.yaml: Helm chart configuration
//   - manifests/<customization>.yaml: Optional customization manifest (if specified)
//   - README.md: Deployment documentation with prerequisites
//   - checksums.txt: SHA256 checksums for verification
//
// # Usage
//
// The bundler is registered in the global bundler registry and can be invoked
// via the CLI or programmatically:
//
//	bundler := skyhook.NewBundler(config)
//	result, err := bundler.Make(ctx, recipe, outputDir)
//
// Or through the bundler framework:
//
//	cnsctl bundle --recipe recipe.yaml --bundlers skyhook-operator --output ./bundles
//
// # Configuration Extraction
//
// The bundler extracts configuration from recipe measurements:
//   - K8s image subtype: Skyhook version
//   - K8s config subtype: Optimization settings (NUMA, huge pages, etc.)
//   - OS subtype: Current kernel parameters for comparison
//   - GPU subtype: GPU-specific optimization requirements
//
// # Skyhook CR
//
// The generated Skyhook custom resource defines:
//   - Target node selectors
//   - Kernel parameter settings
//   - Service configuration overrides
//   - Validation and rollback policies
//
// # Templates
//
// Templates are embedded in the binary using go:embed and rendered with Go's
// text/template package. Templates support:
//   - Workload-specific optimizations (training vs inference)
//   - Hardware-specific settings per GPU architecture
//   - Cloud provider-specific configurations
//
// # Prerequisites
//
// Before deploying Skyhook:
//   - Kubernetes 1.22+ cluster
//   - Skyhook Operator installed
//   - Node-level access for kernel tuning
package skyhook
