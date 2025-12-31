# Cloud Native Stack

**Production-grade, validated configurations for GPU-accelerated Kubernetes at scale.**

Cloud Native Stack (CNS) delivers **opinionated, battle-tested blueprints** for deploying AI workloads on GPU infrastructure. Rather than generic recommendations, CNS provides **validated configurations** proven in production environments, ensuring **reproducible outcomes** across diverse platforms (EKS, GKE, AKS, self-managed).

### Why Cloud Native Stack?

**🎯 Opinionated Best Practices** – Eliminates guesswork with prescriptive configurations based on NVIDIA's extensive field experience and customer deployments.

**✅ Validated & Production-Ready** – Every configuration is tested against real hardware (H100, GB200, A100, L40) and production workloads (training, inference).

**🔁 Reproducible Deployments** – Deterministic recipe generation ensures consistent results across environments, reducing "works on my cluster" issues.

**⚡ Three-Step Workflow** – Systematic approach from discovery to deployment:
1. **Snapshot** – Capture current system configuration (OS, kernel, K8s, GPU)
2. **Recipe** – Generate optimized configuration recommendations based on your environment
3. **Bundle** – Create ready-to-deploy artifacts (Helm values, manifests, scripts)

### Components

- **CLI (`eidos`)** – Command-line tool for snapshot capture, recipe generation, and bundle creation
- **API Server** – REST API (https://cns.dgxc.io) for integration with CI/CD pipelines and automation workflows  
- **Agent** – Kubernetes Job for automated, in-cluster snapshot collection and validation

**Note**: Documentation for the previous version (manual installation guides, playbooks, and platform-specific optimizations) is located in [docs/v1](docs/v1).

## Documentation

The documentation is organized by persona to help you find what you need quickly. Whether you're deploying GPU infrastructure, contributing code, or integrating CNS into automation pipelines, start with the section that matches your role.

### For Users

You're a platform engineer, MLOps practitioner, or cluster administrator responsible for deploying and operating GPU-accelerated Kubernetes clusters. You need practical guides to get CNS running and validated configurations for your specific hardware and workload requirements.

Get started with installing and using Cloud Native Stack:

- **[Installation Guide](docs/user-guide/installation.md)** – Install the eidos CLI (automated script, manual, or build from source)
- **[CLI Reference](docs/user-guide/cli-reference.md)** – Complete command reference with examples
- **[Agent Deployment](docs/user-guide/agent-deployment.md)** – Deploy the Kubernetes agent for automated snapshots

### For Developers

You're a software engineer looking to contribute code, extend functionality, or understand CNS internals. You need development setup instructions, architecture documentation, and guidelines for adding new features like bundlers or collectors.

Learn how to contribute and understand the architecture:

- **[Contributing Guide](CONTRIBUTING.md)** – Development setup, testing, and PR process
- **[Architecture Overview](docs/architecture/README.md)** – System design and components
- **[Bundler Development](docs/architecture/bundler-development.md)** – How to create new bundlers
- **[Data Architecture](docs/architecture/data.md)** – Recipe data model and query matching

### For Integrators

You're an automation engineer, DevOps specialist, or platform architect integrating CNS into CI/CD pipelines, GitOps workflows, or infrastructure-as-code systems. You need API documentation, data schemas, and patterns for programmatic interaction with CNS components.

Integrate Cloud Native Stack into your infrastructure automation:

- **[API Reference](docs/integration/api-reference.md)** – REST API endpoints and usage examples
- **[Data Flow](docs/integration/data-flow.md)** – Understanding snapshots, recipes, and bundles
- **[Automation Guide](docs/integration/automation.md)** – CI/CD integration patterns
- **[Kubernetes Deployment](docs/integration/kubernetes-deployment.md)** – Self-hosted API server setup

### Additional Resources

The manual installation guides for platform-specific deployment and optimization guides:

- **[Installation Guides](docs/v1/install-guides)** – Step-by-step setup for various platforms
- **[Playbooks](docs/v1/playbooks)** – Ansible automation for CNS deployment
- **[Optimizations](docs/v1/optimizations)** – Hardware-specific performance tuning
- **[Troubleshooting](docs/v1/troubleshooting)** – Common issues and solutions
- **[Full Documentation](docs/v1/README.md)** – Complete legacy documentation

## Support

- **Security**: [Project and Artifact Security](docs/SECURITY.md)
- **Releases**: [GitHub Releases](https://github.com/NVIDIA/cloud-native-stack/releases)
- **Issues**: [GitHub Issues](https://github.com/NVIDIA/cloud-native-stack/issues)
- **Questions**: Open a discussion or issue on GitHub
