// Package cli implements the command-line interface for the Cloud Native Stack (CNS) cnsctl tool.
//
// # Overview
//
// The cnsctl CLI provides commands for the four-stage workflow: capturing system snapshots,
// generating configuration recipes, validating constraints, and creating deployment bundles.
// It is designed for cluster administrators and SREs managing NVIDIA GPU infrastructure.
//
// # Commands
//
// snapshot - Capture system configuration (Step 1):
//
//	cnsctl snapshot [--output FILE] [--format yaml|json|table]
//	cnsctl snapshot --output cm://namespace/configmap-name  # ConfigMap output
//	cnsctl snapshot --deploy-agent --namespace gpu-operator  # Agent deployment
//
// Captures a comprehensive snapshot of the current system including CPU/GPU settings,
// kernel parameters, systemd services, and Kubernetes configuration. Supports file,
// stdout, and Kubernetes ConfigMap output.
//
// recipe - Generate configuration recipes (Step 2):
//
//	cnsctl recipe --os ubuntu --osv 24.04 --service eks --gpu h100 --intent training
//	cnsctl recipe --snapshot system.yaml --intent inference --output recipe.yaml
//	cnsctl recipe -s cm://namespace/snapshot -o cm://namespace/recipe  # ConfigMap I/O
//
// Generates optimized configuration recipes based on either:
//   - Specified environment parameters (OS, service, GPU, intent)
//   - Existing system snapshot (analyzes snapshot to extract parameters)
//
// validate - Validate recipe constraints (Step 3):
//
//	cnsctl validate --recipe recipe.yaml --snapshot snapshot.yaml
//	cnsctl validate -r recipe.yaml -s cm://gpu-operator/cns-snapshot
//	cnsctl validate -r recipe.yaml -s cm://ns/snapshot --fail-on-error
//
// Validates recipe constraints against actual measurements from a snapshot.
// Supports version comparisons (>=, <=, >, <), equality (==, !=), and exact match.
// Use --fail-on-error for CI/CD pipelines (non-zero exit on failures).
//
// bundle - Create deployment bundles (Step 4):
//
//	cnsctl bundle --recipe recipe.yaml --output ./bundles
//	cnsctl bundle -r recipe.yaml --bundlers gpu-operator,network-operator -o ./bundles
//	cnsctl bundle -r recipe.yaml --set gpuoperator:driver.version=580.86.16
//
// Generates deployment artifacts (Helm values, manifests, scripts) from recipes.
// Supports multiple bundlers: gpu-operator, network-operator, cert-manager,
// nvsentinel, skyhook.
//
// # Global Flags
//
//	--output, -o   Output file path (default: stdout)
//	--format, -t   Output format: yaml, json, table (default: yaml)
//	--debug        Enable debug logging
//	--log-json     Output logs in JSON format
//	--help, -h     Show command help
//	--version, -v  Show version information
//
// # Output Formats
//
// YAML (default):
//   - Human-readable, preserves structure
//   - Suitable for version control
//
// JSON:
//   - Machine-parseable, compact
//   - Suitable for programmatic consumption
//
// Table:
//   - Hierarchical text representation
//   - Suitable for terminal viewing
//
// # Usage Examples
//
// Complete workflow:
//
//	cnsctl snapshot --output snapshot.yaml
//	cnsctl recipe --snapshot snapshot.yaml --intent training --output recipe.yaml
//	cnsctl validate --recipe recipe.yaml --snapshot snapshot.yaml
//	cnsctl bundle --recipe recipe.yaml --output ./bundles
//
// ConfigMap-based workflow:
//
//	cnsctl snapshot -o cm://gpu-operator/cns-snapshot
//	cnsctl recipe -s cm://gpu-operator/cns-snapshot -o cm://gpu-operator/cns-recipe
//	cnsctl validate -r cm://gpu-operator/cns-recipe -s cm://gpu-operator/cns-snapshot
//	cnsctl bundle -r cm://gpu-operator/cns-recipe -o ./bundles
//
// Generate recipe for Ubuntu 24.04 on EKS with H100 GPUs:
//
//	cnsctl recipe --os ubuntu --osv 24.04 --service eks --gpu h100 --intent training
//
// Override bundle values at generation time:
//
//	cnsctl bundle -r recipe.yaml --set gpuoperator:gds.enabled=true -o ./bundles
//
// # Environment Variables
//
//	LOG_LEVEL              Set logging verbosity (debug, info, warn, error)
//	NODE_NAME              Override node name for Kubernetes collection
//	KUBERNETES_NODE_NAME   Fallback node name if NODE_NAME not set
//	HOSTNAME               Final fallback for node name
//	KUBECONFIG             Path to kubeconfig file
//
// # Exit Codes
//
//	0  Success
//	1  General error (invalid arguments, execution failure)
//	2  Context canceled or timeout
//
// # Architecture
//
// The CLI uses the urfave/cli/v3 framework and delegates to specialized packages:
//   - pkg/snapshotter - System snapshot collection
//   - pkg/recipe - Recipe generation from queries or snapshots
//   - pkg/bundler - Bundle orchestration and generation
//   - pkg/component - Individual bundler implementations
//   - pkg/serializer - Output formatting (including ConfigMap)
//   - pkg/logging - Structured logging
//
// Version information is embedded at build time using ldflags:
//
//	go build -ldflags="-X 'github.com/NVIDIA/cloud-native-stack/pkg/cli.version=1.0.0'"
package cli
