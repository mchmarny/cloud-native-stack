// Package dradriver implements bundle generation for NVIDIA DRA Driver.
//
// The DRA Driver manages GPU and ComputeDomain resources in Kubernetes clusters by automating deployment
// and lifecycle management of the DRA Driver controller and kubelet plugins.
//
// # Bundle Structure
//
// Generated bundles include:
//   - values.yaml: Helm chart configuration
//   - scripts/install.sh: Automated installation script
//   - scripts/uninstall.sh: Cleanup script
//   - README.md: Deployment documentation with prerequisites
//   - checksums.txt: SHA256 checksums for verification
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
//	cnsctl bundle --recipe recipe.yaml --bundlers dra-driver --output ./bundles
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
//   - Cloud provider-specific optimizations
//
// # Validation
//
// The bundler performs validation to ensure:
//   - Recipe contains required K8s image measurements
//   - Configuration is internally consistent
//   - All required template data is available
//
// # Parallel Execution
//
// When multiple bundlers are registered, they execute in parallel with proper
// synchronization. The BaseBundler helper handles concurrent writes safely.
package dradriver
