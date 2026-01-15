/*
Package deployer provides deployment artifact generation for CNS bundles.

The deployer package enables pluggable deployment methods for bundle artifacts.
Each deployer type generates appropriate deployment manifests, scripts, and
documentation for a specific deployment workflow (manual scripts, ArgoCD, Flux, etc.).

# Architecture

Deployers follow the registry pattern similar to the bundler package:
  - Each deployer implements the Deployer interface
  - Deployers self-register via init() functions
  - A central registry manages available deployers
  - Factory functions create deployer instances

# Usage

The deployer is selected via the --deployer flag in the bundle command:

	cnsctl bundle -r recipe.yaml --deployer script  # Default: manual deployment
	cnsctl bundle -r recipe.yaml --deployer argocd  # ArgoCD GitOps
	cnsctl bundle -r recipe.yaml --deployer flux    # Flux GitOps

# Supported Deployers

  - script: Generates shell scripts and basic README for manual deployment (default)
  - argocd: Generates ArgoCD Application manifests for GitOps deployment
  - flux: Generates Flux Kustomization resources for GitOps deployment

# Adding New Deployers

To add a new deployer:

 1. Create a new package under pkg/deployer/<name>/
 2. Implement the Deployer interface
 3. Create a factory function: func NewDeployer() Deployer
 4. Self-register in init(): registry.MustRegister(types.DeployerType<Name>, NewDeployer)

Example:

	package helm

	import (
		"context"
		"github.com/NVIDIA/cloud-native-stack/pkg/deployer/registry"
		"github.com/NVIDIA/cloud-native-stack/pkg/deployer/types"
	)

	func init() {
		registry.MustRegister(types.DeployerTypeHelm, NewDeployer)
	}

	type Deployer struct{}

	func NewDeployer() Deployer {
		return &Deployer{}
	}

	func (d *Deployer) Generate(ctx context.Context, ...) (*result.Artifacts, error) {
		// Implementation
	}
*/
package deployer
