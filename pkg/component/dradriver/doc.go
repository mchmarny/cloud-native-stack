// Package dradriver implements bundle generation for NVIDIA DRA Driver.
//
// The DRA Driver manages GPU and ComputeDomain resources in Kubernetes clusters
// using Kubernetes Dynamic Resource Allocation (DRA) for GPU scheduling. It provides:
//   - DRA Driver controller for resource claim management
//   - Kubelet plugin for node-level GPU allocation
//   - ComputeDomain resource support
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
//   - Default Helm chart (nvidia/nvidia-dra-driver-gpu)
//   - Value override key mapping (dradriver, nvidia-dra-driver-gpu)
//
// # Usage
//
// The bundler is registered in the global bundler registry and can be invoked
// via the CLI or programmatically:
//
//	bundler := dradriver.NewBundler(config)
//	result, err := bundler.Make(ctx, recipe, outputDir)
//
// Or through the bundler framework:
//
//	cnsctl bundle --recipe recipe.yaml --bundlers nvidia-dra-driver-gpu --output ./bundles
//
// # Configuration Extraction
//
// The bundler extracts values from recipe component references. The recipe's
// matchedRules field indicates which overlays were applied.
package dradriver
