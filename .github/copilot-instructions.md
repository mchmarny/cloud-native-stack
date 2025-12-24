# Copilot Instructions for NVIDIA Cloud Native Stack

## Project Overview
NVIDIA Cloud Native Stack (CNS) is a reference architecture and toolkit for deploying cloud-native workloads on NVIDIA GPUs, using Kubernetes, Helm, and NVIDIA Operators. It supports Ubuntu/RHEL platforms and is designed for both manual and automated deployments.

## Architecture & Key Components
- **api/**: API definitions (YAML) for CNS services.
- **cmd/**: Entrypoints for CLI and server binaries.
- **pkg/**: Core Go packages for CLI, client, collector, logging, measurement, recommendation, server, snapshotter, and versioning.
- **docs/**: Install guides, playbooks (Ansible), optimizations, troubleshooting, and topologies.
- **deployments/**: Kubernetes manifests for CNS components.
- **examples/**: Example configurations for supported hardware.
- **tools/**: Utility scripts for build and release workflows.

## Developer Workflows
- **Build:** Use `make` for common build tasks. Example: `make build`.
- **Test:** Run Go tests with `make test` or `go test ./...`.
- **Deploy:** Use playbooks in `docs/playbooks/` and manifests in `deployments/eidos/`.
- **Install:** Run `bash docs/playbooks/setup.sh install` for automated CNS installation.
- **Hosts File:** Edit `docs/playbooks/hosts` to specify master/worker nodes for Ansible.

## Patterns & Conventions
- **Go Modules:** All Go code uses modules (`go.mod`).
- **CLI Entrypoints:** Located in `cmd/eidos/main.go` and `cmd/server/main.go`.
- **Collectors:** Extendable via `pkg/collector/` with factory pattern.
- **Logging:** Centralized in `pkg/logging/logger.go`.
- **Recommendations:** Types and logic in `pkg/recommendation/`.
- **Playbooks:** YAML-based automation in `docs/playbooks/`.
- **Optimizations:** Hardware-specific tuning in `docs/optimizations/`.

## Integration Points
- **Kubernetes:** Manifests and playbooks deploy CNS on K8s clusters.
- **NVIDIA Operators:** GPU, Network, NIM, and Nsight Operators are integrated via Helm and manifests.
- **MicroK8s:** Supported as a lightweight K8s option.
- **Monitoring/Storage/LoadBalancer:** Prometheus, NFS, MetalLB, and others are configured via playbooks.

## Troubleshooting & Support
- **Troubleshooting:** See `docs/troubleshooting/README.md` and linked guides for common issues (e.g., DNS).
- **Feedback:** Open issues on GitHub for questions or bugs.

## Examples
- To add a new collector, implement in `pkg/collector/` and register in the factory.
- To update deployment manifests, edit files in `deployments/eidos/`.
- To customize installation, modify playbooks in `docs/playbooks/` and hosts in `docs/playbooks/hosts`.

---
For further details, see the main [README.md](../README.md) and [docs/playbooks/README.md](../docs/playbooks/readme.md).
