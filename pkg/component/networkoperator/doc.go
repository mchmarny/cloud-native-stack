// Package networkoperator implements bundle generation for NVIDIA Network Operator.
//
// The Network Operator enables advanced networking features on GPU-accelerated
// Kubernetes clusters including:
//   - RDMA (Remote Direct Memory Access) for high-performance networking
//   - SR-IOV (Single Root I/O Virtualization) for network device virtualization
//   - OFED (OpenFabrics Enterprise Distribution) driver deployment
//   - Host Device Plugin for exposing Mellanox/ConnectX NICs
//   - NVIDIA IPAM (IP Address Management) plugin
//   - Multus CNI for multiple network interfaces
//
// # Bundle Structure
//
// Generated bundles include:
//   - values.yaml: Helm chart configuration
//   - checksums.txt: SHA256 checksums for verification
//
// # Implementation
//
// This bundler uses the generic bundler framework from [internal.ComponentConfig]
// and [internal.MakeBundle]. The componentConfig variable defines:
//   - Default Helm repository (https://helm.ngc.nvidia.com/nvidia)
//   - Default Helm chart (nvidia/network-operator)
//   - Value override key mapping (networkoperator)
//
// This is a minimal bundler implementation with no custom metadata or manifest
// generation, demonstrating the simplest use of the generic framework.
//
// # Usage
//
// The bundler is registered in the global bundler registry and can be invoked
// via the CLI or programmatically:
//
//	bundler := networkoperator.NewBundler(config)
//	result, err := bundler.Make(ctx, recipe, outputDir)
//
// Or through the bundler framework:
//
//	cnsctl bundle --recipe recipe.yaml --bundlers network-operator --output ./bundles
//
// # Configuration Extraction
//
// The bundler extracts values from recipe component references. Configuration
// includes RDMA, SR-IOV, IPAM, and Multus settings.
package networkoperator
