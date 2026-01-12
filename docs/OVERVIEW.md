# Cloud Native Stack: An Overview

NVIDIA Cloud Native Stack (CNS) is a suite of tooling designed to automate the complexity of deploying GPU-accelerated Kubernetes infrastructure. By moving away from static documentation and toward automated configuration generation, CNS ensures that AI/ML workloads run on infrastructure that is validated, optimized, and secure.

## Why CNS?

Deploying high-performance AI infrastructure is historically complex. Administrators must navigate a "matrix" of dependencies, ensuring compatibility between the Operating System, Kubernetes version, GPU drivers, and container runtimes.

### The Challenge: The "Old Way"
Previously, administrators relied on static documentation and manual installation guides. This approach presented several significant challenges:
*   **Complexity:** Administrators had to manually track compatibility matrices across dozens of components (e.g., matching a specific GPU Operator version to a specific driver and K8s version).
*   **Human Error:** Manual copy-pasting of commands and flags often led to configuration drift or broken deployments.
*   **Documentation Drift:** Static guides (like Markdown files) quickly become outdated as new software versions are released, leading to "documentation drift".
*   **Lack of Optimization:** Generic installation guides rarely account for specific hardware differences (e.g., H100 vs. GB200) or workload intents (Training vs. Inference).

### The Solution: Automated Approach
CNS replaces manual interpretation of documentation with a **automated approach**. It treats infrastructure configuration as code, providing a deterministic engine that generates the exact artifacts needed for a specific environment.

**Key Benefits:**
1.  **Deterministic & Validated:** The system guarantees that the inputs (your system state) always produce the same valid outputs, tested against NVIDIA hardware.
2.  **Hardware-Aware Optimization:** CNS detects the specific GPU type (e.g., H100, A100, GB200) and OS to apply hardware-specific tuning automatically.
3.  **Speed:** Deployment preparation drops from hours of reading and configuration to minutes of automated generation.
4.  **Supply Chain Security:** All artifacts are backed by SLSA Build Level 3 attestations and Software Bill of Materials (SBOMs), ensuring the software stack is secure and verifiable.

## How CNS Works

CNS simplifies operations through a logical four-stage workflow handled by the `eidos` command-line tool. This workflow transforms a raw system state into a deployable package.

### Step 1: Snapshot (Capture Reality)

Before configuring anything, CNS needs to understand the environment.
*   **What it does:** The system captures the state of the OS, SystemD services, Kubernetes version, and GPU hardware.
*   **How it helps:** It eliminates guesswork. Instead of assuming what hardware is present, CNS measures it directly using the CLI or a Kubernetes Agent.
*   **Automation:** The agent can run as a Kubernetes Job, writing the snapshot directly to a ConfigMap, enabling fully automated auditing without manual intervention.

### Step 2: Recipe (Generate Recommendations)

Once the system state is known, CNS generates a "Recipe"—a set of configuration recommendations.
*   **What it does:** It matches the snapshot against a database of validated rules (overlays). It selects the correct driver versions, kernel modules, and settings for that specific environment.
*   **Intent-Based Tuning:** Users can specify an "Intent" (e.g., `training` or `inference`). CNS adjusts the recipe to optimize for throughput (training) or latency (inference).
*   **How it helps:** It ensures version compatibility and applies expert-level optimizations automatically, acting as a dynamic compatibility matrix.

### Step 3: Validate (Check Compatibility)

Before deploying, CNS can validate that a target cluster meets the recipe requirements.
*   **What it does:** It compares recipe constraints (version requirements, configuration settings) against actual measurements from a cluster snapshot.
*   **Constraint Types:** Supports version comparisons (`>=`, `<=`, `>`, `<`), equality (`==`, `!=`), and exact match for configuration values.
*   **How it helps:** It catches compatibility issues before deployment, preventing failed rollouts and configuration drift. Ideal for CI/CD pipelines with `--fail-on-error` flag.

### Step 4: Bundle (Create Artifacts)

Finally, CNS converts the abstract Recipe into concrete deployment files.
*   **What it does:** It generates a "Bundle" containing Helm values, Kubernetes manifests, installation scripts, and a custom README.
*   **How it helps:** Users receive ready-to-run scripts and manifests. For example, it generates a custom `install.sh` script that pre-validates the environment before running Helm commands.
*   **Parallel Execution:** Multiple "Bundlers" (e.g., GPU Operator, Network Operator) can run simultaneously to generate a full stack configuration in seconds.

## Key Capabilities

### Kubernetes-Native Integration
CNS is designed to work natively within Kubernetes.
*   **ConfigMap Support:** You don't need to manage local files. You can read and write Snapshots and Recipes directly to Kubernetes ConfigMaps using the URI format `cm://namespace/name`.
*   **No Persistent Volumes:** The automated Agent writes data directly to the Kubernetes API, simplifying deployment in restricted environments.

### Integration & Automation
*   **CI/CD Ready:** The `eidos` CLI and API server are built for pipelines. Teams can use CNS to detect "Configuration Drift" by periodically taking snapshots and comparing them to a baseline.
*   **API Server:** For programmatic access, CNS provides a production-ready HTTP REST API to generate recipes dynamically.

### Security
CNS prioritizes trust in the software supply chain.
*   **Verifiable Builds:** Every release includes provenance data showing exactly how and where it was built (SLSA Level 3).
*   **SBOMs:** Complete inventories of all dependencies are provided for both binaries and container images, enabling automated vulnerability scanning.
