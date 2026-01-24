/*
Copyright © 2025 NVIDIA Corporation
SPDX-License-Identifier: Apache-2.0
*/

// Package helm generates Helm umbrella charts from recipe results.
//
// Generates umbrella charts with dependencies for deploying multiple components:
//
//   - Chart.yaml with component dependencies
//   - Combined values.yaml for all components
//   - README.md with deployment instructions
//   - checksums.txt for verification (optional)
//
// Usage:
//
//	generator := helm.NewGenerator()
//	input := &helm.GeneratorInput{
//	    RecipeResult:     recipeResult,
//	    ComponentValues:  componentValues,
//	    Version:          "1.0.0",
//	    IncludeChecksums: true,
//	}
//	output, err := generator.Generate(ctx, input, "/path/to/output")
package helm
