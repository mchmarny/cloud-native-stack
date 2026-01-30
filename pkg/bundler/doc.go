// Copyright (c) 2025, NVIDIA CORPORATION.  All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

/*
Package bundler provides orchestration for generating deployment bundles from recipes.

The bundler package generates deployment-ready artifacts (Helm umbrella charts or
ArgoCD applications) from recipe configurations. Component configuration is loaded
from the declarative component registry (pkg/recipe/data/registry.yaml).

# Architecture

  - DefaultBundler: Generates Helm umbrella charts or ArgoCD applications
  - Component Registry: Declarative configuration in pkg/recipe/data/components.yaml
  - Deployers: Helm (default) and ArgoCD output formats
  - result.Output: Aggregated generation results

# Quick Start

	b, err := bundler.New()
	output, err := b.Make(ctx, recipeResult, "./bundle")
	fmt.Printf("Generated: %d files\n", output.TotalFiles)

With options:

	cfg := config.NewConfig(
	    config.WithDeployer(config.DeployerHelm),
	    config.WithIncludeChecksums(true),
	)
	b, err := bundler.New(bundler.WithConfig(cfg))

# Supported Components

Components are defined in pkg/recipe/data/registry.yaml:

  - gpu-operator: NVIDIA GPU Operator
  - network-operator: NVIDIA Network Operator
  - nvidia-dra-driver-gpu: NVIDIA DRA Driver
  - cert-manager: Certificate Manager
  - nvsentinel: NVSentinel
  - skyhook-operator: Skyhook node optimization

# Output Formats

Helm (default):
  - Chart.yaml: Helm umbrella chart with dependencies
  - values.yaml: Combined values for all components
  - README.md: Deployment instructions
  - recipe.yaml: Copy of the input recipe
  - templates/: Custom manifest templates (if any)

ArgoCD:
  - app-of-apps.yaml: Parent ArgoCD Application
  - <component>/application.yaml: ArgoCD Application per component
  - <component>/values.yaml: Values for each component

# Configuration

	cfg := config.NewConfig(
	    config.WithDeployer(config.DeployerHelm),
	    config.WithIncludeReadme(true),
	    config.WithSystemNodeSelector(map[string]string{"node-role": "system"}),
	)
	b, err := bundler.New(bundler.WithConfig(cfg))

# Adding New Components

To add a new component, add an entry to pkg/recipe/data/registry.yaml.
No Go code is required.

Helm Component Example:

  - name: my-component
    displayName: My Component
    valueOverrideKeys: [mycomponent]
    helm:
    defaultRepository: https://charts.example.com
    defaultChart: example/my-component
    nodeScheduling:
    system:
    nodeSelectorPaths: [operator.nodeSelector]

Kustomize Component Example:

  - name: my-kustomize-app
    displayName: My Kustomize App
    valueOverrideKeys: [mykustomize]
    kustomize:
    defaultSource: https://github.com/example/my-app
    defaultPath: deploy/production
    defaultTag: v1.0.0

Note: A component must have either 'helm' OR 'kustomize' configuration, not both.

See https://github.com/NVIDIA/cloud-native-stack for more information.
*/
package bundler
