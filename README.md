# Cloud Native Stack

Cloud Native Stack (CNS) provides validated configuration guidance for deploying GPU-accelerated Kubernetes infrastructure. It captures known-good combinations of software, configuration, and system requirements and makes them consumable as documentation and generated deployment artifacts.

## Why We Built This

Running NVIDIA-accelerated Kubernetes clusters reliably is hard. Small differences in kernel versions, drivers, container runtimes, operators, and Kubernetes releases can cause failures that are difficult to diagnose and expensive to reproduce.

Historically, this knowledge has lived in internal validation pipelines, playbooks, and tribal knowledge. Cloud Native Stack exists to externalize that experience. Its goal is to make validated configurations visible, repeatable, and reusable across environments.

## What Cloud Native Stack Is (and Is Not)

Cloud Native Stack is a **source of validated configuration knowledge** for NVIDIA-accelerated Kubernetes environments.

It **is**:
- A curated set of tested and validated component combinations
- A reference for how NVIDIA-accelerated Kubernetes clusters are expected to be configured
- A foundation for generating reproducible deployment artifacts
- Designed to integrate with existing provisioning, CI/CD, and GitOps workflows

It **is not**:
- A Kubernetes distribution
- A cluster provisioning or lifecycle management system
- A managed control plane or hosted service
- A replacement for cloud provider or OEM platforms

---
### Note on previous versions**  
> Earlier versions of Cloud Native Stack focused primarily on manual installation guides and playbooks. Those materials remain available under [`/~archive/cns-v1`](/~archive/cns-v1/). The current repository reflects a transition toward structured configuration data and generated artifacts.
---

## How It Works

Cloud Native Stack separates **validated configuration knowledge** from **how that knowledge is consumed**.

- Human-readable documentation lives under `docs/`.
- Version-locked configuration definitions (“recipes”) capture known-good system states.
- Those definitions can be rendered into concrete artifacts such as Helm values, Kubernetes manifests, or install scripts.- Recipes can be validated against actual system configurations to verify compatibility.
This separation allows the same validated configuration to be applied consistently across different environments and automation systems.

*For example, a configuration validated for H100 on Ubuntu 22.04 with Kubernetes 1.29 can be rendered into Helm values and manifests suitable for use in an existing GitOps pipeline.*

## Get Started

> Some tooling and APIs are under active development; documentation reflects current and near-term capabilities.

### Quick Start

Get started quickly with CNS:
1. Review the documentation under `docs/` to understand supported platforms and required components.
2. Identify your target environment:
   - GPU architecture
   - Operating system and kernel
   - Kubernetes distribution and version
   - Workload intent (for example, training or inference)
3. Apply the validated configuration guidance using your existing tools (Helm, kubectl, CI/CD, or GitOps).
4. Validate and iterate as platforms and workloads evolve.

### Get Started by Use Case

These use cases reflect common ways teams interact with Cloud Native Stack.

<details>
<summary><strong>Platform and Infrastructure Operators</strong></summary>

You are responsible for deploying and operating GPU-accelerated Kubernetes clusters. 
- **[Installation Guide](docs/user-guide/installation.md)** – Install the cnsctl CLI (automated script, manual, or build from source)
- **[CLI Reference](docs/user-guide/cli-reference.md)** – Complete command reference with examples
- **[API Reference](docs/user-guide/api-reference.md)** – Complete API reference with examples
- **[Agent Deployment](docs/user-guide/agent-deployment.md)** – Deploy the Kubernetes agent to get automated configuration snapshots
</details>

<details>
<summary><strong>Developers and Contributors</strong></summary>

You are contributing code, extending functionality, or working on CNS internals. 

- **[Contributing Guide](CONTRIBUTING.md)** – Development setup, testing, and PR process
- **[Architecture Overview](docs/architecture/README.md)** – System design and components
- **[Bundler Development](docs/architecture/component.md)** – How to create new bundlers
- **[Data Architecture](docs/architecture/data.md)** – Recipe data model and query matching
</details>

<details>
<summary><strong>Integrators and Automation Engineers</strong></summary>

You are integrating CNS into CI/CD pipelines, GitOps workflows, or a larger product or service. 

- **[API Reference](docs/integration/api-reference.md)** – REST API endpoints and usage examples
- **[Data Flow](docs/integration/data-flow.md)** – Understanding snapshots, recipes, and bundles
- **[Automation Guide](docs/integration/automation.md)** – CI/CD integration patterns
- **[Kubernetes Deployment](docs/integration/kubernetes-deployment.md)** – Self-hosted API server setup
</details>

## Project Structure

- `api/` — OpenAPI specifications for the REST API
- `cmd/` — Entry points for CLI (`cnsctl`) and API server (`cnsd`)
- `deployments/` — Kubernetes manifests for agent deployment
- `docs/` — User-facing documentation, guides, and architecture docs
- `examples/` — Example snapshots, recipes, and comparisons
- `infra/` — Infrastructure as code (Terraform) for deployments
- `pkg/` — Core Go packages (collectors, recipe engine, bundlers, serializers)
- `tools/` — Build scripts, E2E testing, and utilities
- `~archive/` — Archived v1 installation guides and playbooks

## Documentation & Resources

- **[Documentation](/docs)** – Documentation, guides, and examples.
- **[Roadmap](ROADMAP.md)** – Feature priorities and development timeline
- **[Transition](docs/MIGRATION.md)** - Migration to CLI/API-based bundle generation
- **[Security](SECURITY.md)** - Security-related resources 
- **[Releases](https://github.com/NVIDIA/cloud-native-stack/releases)** - Binaries, SBOMs, and other artifacts
- **[Issues](https://github.com/NVIDIA/cloud-native-stack/issues)** - Bugs, feature requests, and questions

## Contributing

Contributions are welcome. See [contributing](/CONTRIBUTING.md) for development setup, contribution guidelines, and the pull request process.