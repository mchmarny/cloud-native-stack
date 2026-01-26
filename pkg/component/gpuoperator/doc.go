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
//   - checksums.txt: SHA256 checksums for verification
//
// # Implementation
//
// This bundler uses the generic bundler framework from [internal.ComponentConfig]
// and [internal.MakeBundle]. The componentConfig variable defines:
//   - Node selector and toleration paths for operator and daemonset workloads
//   - Default Helm repository (https://helm.ngc.nvidia.com/nvidia)
//   - CustomManifestFunc for generating DCGM exporter and kernel module manifests
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
// The bundler extracts values from recipe component references. The recipe's
// matchedRules field indicates which overlays were applied, enabling
// workload-specific optimizations (training vs inference).
//
// # Custom Manifests
//
// The bundler generates additional manifests via CustomManifestFunc:
//   - DCGM Exporter ConfigMap (when dcgm-exporter.arguments contains custom args)
//   - Kernel Module Parameters ConfigMap (for GB200 accelerator tuning)
//
// # Templates
//
// Templates are embedded in the binary using go:embed and rendered with Go's
// text/template package. Templates support:
//   - Conditional sections based on enabled features
//   - Version-specific configurations
//   - Hardware-specific optimizations
//   - Workload intent tuning (training/inference)
package gpuoperator
