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
//   - README.md: Deployment documentation with prerequisites and instructions
//   - checksums.txt: SHA256 checksums for verification
//
// # Usage
//
// The bundler is registered in the global bundler registry and can be invoked
// via the CLI or programmatically:
//
//	bundler := networkoperator.NewBundler(config)
//	result, err := bundler.Make(ctx, recipe, outputDir)
//
// # Configuration Extraction
//
// The bundler extracts configuration from recipe measurements:
//   - K8s image subtype: Network Operator version, OFED version
//   - K8s config subtype: RDMA, SR-IOV, IPAM, Multus settings
//
// # Templates
//
// Templates are embedded in the binary using go:embed and rendered with Go's
// text/template package. Templates support conditional sections based on
// enabled features.
package networkoperator
