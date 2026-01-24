// Package gpuoperator implements bundle generation for NVIDIA GPU Operator.
//
// The GPU Operator manages GPU resources in Kubernetes clusters by automating deployment
// and lifecycle management of:
//   - GPU drivers (containerized NVIDIA driver)
//   - Device Plugin (exposes GPUs to Kubernetes scheduler)
//   - DCGM Exporter (GPU metrics for Prometheus)
//   - MIG Manager (Multi-Instance GPU partitioning)
//   - Node Feature Discovery (GPU hardware detection)
//   - Container Toolkit (nvidia-container-runtime)
//
// # Bundle Structure
//
// Generated bundles include:
//   - values.yaml: Helm chart configuration
//   - manifests/: Kubernetes manifests (DCGM Exporter ConfigMap, Kernel Module Params)
//   - README.md: Deployment documentation with prerequisites and instructions
//   - checksums.txt: SHA256 checksums for verification
//
// # Usage
//
// The bundler is registered in the global bundler registry and can be invoked
// via the CLI or programmatically:
//
//	bundler := gpuoperator.NewBundler(config)
//	result, err := bundler.Make(ctx, recipe, outputDir)
//
// Or through the bundler framework:
//
//	cnsctl bundle --recipe recipe.yaml --bundlers gpu-operator --output ./bundles
//
// # Configuration Extraction
//
// The bundler extracts configuration from recipe measurements:
//   - K8s image subtype: GPU Operator version, driver version, toolkit version
//   - K8s config subtype: MIG mode, secure boot, CDI, vGPU, GDS settings
//   - GPU subtypes: Hardware-specific optimizations
//
// The recipe's matchedRules field indicates which overlays were applied, enabling
// workload-specific optimizations (training vs inference).
//
// # Templates
//
// Templates are embedded in the binary using go:embed and rendered with Go's
// text/template package. Templates support:
//   - Conditional sections based on enabled features
//   - Version-specific configurations
//   - Hardware-specific optimizations
//   - Workload intent tuning (training/inference)
//
// # Validation
//
// The bundler performs validation to ensure:
//   - Recipe contains required K8s image measurements
//   - GPU Operator version is specified
//   - Configuration is internally consistent
//   - All required template data is available
//
// # Parallel Execution
//
// When multiple bundlers are registered, they execute in parallel with proper
// synchronization. The BaseBundler helper handles concurrent writes safely.
package gpuoperator
