// Package cli implements the command-line interface for the Cloud Native Stack (CNS) eidos tool.
//
// # Overview
//
// The eidos CLI provides commands for capturing system snapshots, generating configuration recipes,
// and producing recommendations for optimizing GPU-accelerated Kubernetes clusters. It is designed
// for cluster administrators and SREs managing NVIDIA GPU infrastructure.
//
// # Commands
//
// snapshot - Capture system configuration:
//
//	eidos snapshot [--output FILE] [--format yaml|json|table]
//
// Captures a comprehensive snapshot of the current system including CPU/GPU settings,
// kernel parameters, systemd services, and Kubernetes configuration. Output defaults to
// stdout in YAML format.
//
// recipe - Generate configuration recipes:
//
//	eidos recipe --os ubuntu --osv 24.04 --service eks --gpu h100 --intent training
//
// Generates optimized configuration recipes based on specified environment parameters.
// Supports various OS families (Ubuntu, RHEL), Kubernetes services (EKS, GKE, AKS),
// GPU types, and workload intents (training, inference).
//
// recommend - Generate recommendations from snapshot:
//
//	eidos recommend --snapshot system.yaml --intent training [--output recommendations.yaml]
//
// Analyzes a captured snapshot and generates tailored configuration recommendations
// based on the specified workload intent. Produces a Recipe with optimized settings.
//
// # Global Flags
//
//	--output, -o   Output file path (default: stdout)
//	--format, -t   Output format: yaml, json, table (default: yaml)
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
// Capture snapshot to file:
//
//	eidos snapshot --output system.yaml
//
// Generate recipe for Ubuntu 24.04 on EKS with H100 GPUs:
//
//	eidos recipe --os ubuntu --osv 24.04 --service eks --gpu h100 --intent training --format json
//
// Generate recommendations with context:
//
//	eidos recommend -f system.yaml -i inference --context --output recommendations.yaml
//
// # Environment Variables
//
//	LOG_LEVEL          Set logging verbosity (debug, info, warn, error)
//	NODE_NAME          Override node name for Kubernetes collection
//	KUBERNETES_NODE_NAME  Fallback node name if NODE_NAME not set
//	HOSTNAME           Final fallback for node name
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
//   - pkg/recipe - Recipe generation
//   - pkg/recommender - Snapshot analysis and recommendations
//   - pkg/serializer - Output formatting
//   - pkg/logging - Structured logging
//
// Version information is embedded at build time using ldflags:
//
//	go build -ldflags="-X 'github.com/NVIDIA/cloud-native-stack/pkg/cli.version=1.0.0'"
package cli
