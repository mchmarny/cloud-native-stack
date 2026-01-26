// Package nvsentinel implements bundle generation for NVIDIA NVSentinel.
//
// NVSentinel provides monitoring and telemetry for GPU-accelerated Kubernetes
// clusters, enabling:
//   - Real-time GPU health monitoring and alerting
//   - Performance metrics collection and visualization
//   - Anomaly detection for GPU workloads
//   - Integration with Prometheus and Grafana
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
//   - Default Helm chart (nvidia/nvsentinel)
//   - Default chart version (v0.6.0)
//   - Node selector and toleration paths for workload placement
//
// This is a minimal bundler implementation using the generic framework without
// custom metadata extensions or manifest generation.
//
// # Usage
//
// The bundler is registered in the global bundler registry and can be invoked
// via the CLI or programmatically:
//
//	bundler := nvsentinel.NewBundler(config)
//	result, err := bundler.Make(ctx, recipe, outputDir)
//
// Or through the bundler framework:
//
//	cnsctl bundle --recipe recipe.yaml --bundlers nvsentinel --output ./bundles
//
// # Configuration Extraction
//
// The bundler extracts values from recipe component references including
// monitoring settings and alert thresholds.
//
// # Prerequisites
//
// Before deploying NVSentinel:
//   - Kubernetes 1.22+ cluster
//   - NVIDIA GPU Operator installed
//   - Prometheus Operator (optional, for metrics)
package nvsentinel
