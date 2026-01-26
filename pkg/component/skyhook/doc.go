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
//   - checksums.txt: SHA256 checksums for verification
//
// # Implementation
//
// This bundler uses the generic bundler framework from [internal.ComponentConfig]
// and [internal.MakeBundle]. The componentConfig variable defines:
//   - Default Helm repository (https://nvidia.github.io/skyhook)
//   - Default Helm chart (skyhook)
//   - CustomManifestFunc for generating customization CR manifests
//   - MetadataExtensions for additional template data
//
// # Custom Metadata
//
// This bundler provides custom metadata via MetadataExtensions to include:
//   - HelmChartName: Display name for the chart
//
// These extensions are accessible in templates via {{ .Script.Extensions.HelmChartName }}.
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
// The bundler extracts values from recipe component references including
// optimization settings for NUMA, huge pages, and GPU-specific parameters.
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
