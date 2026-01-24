/*
Copyright © 2025 NVIDIA Corporation
SPDX-License-Identifier: Apache-2.0

Package argocd provides ArgoCD Application generation for Cloud Native Stack recipes.

The argocd package generates ArgoCD Application manifests from RecipeResult objects,
enabling GitOps-based deployment of GPU-accelerated infrastructure components.

# Overview

The package supports the App of Apps pattern, generating:
  - Individual Application manifests for each component
  - An app-of-apps.yaml manifest that manages all applications
  - Values files for Helm chart configuration
  - README with deployment instructions

# Deployment Ordering

Components are deployed in order using ArgoCD sync-waves. The deployment order
is determined by the recipe's DeploymentOrder field. Components are assigned
sync-wave annotations starting from 0.

# Usage

	generator := argocd.NewGenerator()

	input := &argocd.GeneratorInput{
		RecipeResult:     recipeResult,
		ComponentValues:  componentValues,
		Version:          "v0.9.0",
		RepoURL:          "https://github.com/my-org/my-gitops-repo.git",
		IncludeChecksums: true,
	}

	output, err := generator.Generate(ctx, input, "/path/to/output")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Generated %d files (%d bytes)\n", len(output.Files), output.TotalSize)

# Generated Structure

	output/
	├── app-of-apps.yaml           # Parent application
	├── README.md                  # Deployment instructions
	├── checksums.txt              # SHA256 checksums (optional)
	├── cert-manager/
	│   ├── application.yaml       # ArgoCD Application (sync-wave: 0)
	│   └── values.yaml
	├── gpu-operator/
	│   ├── application.yaml       # ArgoCD Application (sync-wave: 1)
	│   └── values.yaml
	└── network-operator/
	    ├── application.yaml       # ArgoCD Application (sync-wave: 2)
	    └── values.yaml

# Configuration

The RepoURL field in GeneratorInput sets the Git repository URL in the
app-of-apps.yaml manifest. If not provided, a placeholder URL is used
that must be updated manually before deployment.
*/
package argocd
