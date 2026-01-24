/*
Copyright © 2025 NVIDIA Corporation
SPDX-License-Identifier: Apache-2.0
*/

package helm

import (
	"context"
	_ "embed"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/template"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/NVIDIA/cloud-native-stack/pkg/bundler/checksum"
	"github.com/NVIDIA/cloud-native-stack/pkg/errors"
	"github.com/NVIDIA/cloud-native-stack/pkg/recipe"
)

//go:embed templates/Chart.yaml.tmpl
var chartTemplate string

//go:embed templates/README.md.tmpl
var readmeTemplate string

// criteriaAny is the wildcard value for criteria fields.
const criteriaAny = "any"

// ChartMetadata represents the metadata for an umbrella Helm chart.
type ChartMetadata struct {
	APIVersion   string       `yaml:"apiVersion"`
	Name         string       `yaml:"name"`
	Description  string       `yaml:"description"`
	Type         string       `yaml:"type"`
	Version      string       `yaml:"version"`
	AppVersion   string       `yaml:"appVersion"`
	Dependencies []Dependency `yaml:"dependencies"`
}

// Dependency represents a Helm chart dependency.
type Dependency struct {
	Name       string `yaml:"name"`
	Version    string `yaml:"version"`
	Repository string `yaml:"repository"`
	Condition  string `yaml:"condition,omitempty"`
}

// GeneratorInput contains all data needed to generate an umbrella chart.
type GeneratorInput struct {
	// RecipeResult contains the recipe metadata and component references.
	RecipeResult *recipe.RecipeResult

	// ComponentValues maps component names to their values.
	// These are collected from individual bundlers.
	ComponentValues map[string]map[string]interface{}

	// Version is the chart version (from CLI/bundler version).
	Version string

	// IncludeChecksums indicates whether to generate a checksums.txt file.
	IncludeChecksums bool
}

// GeneratorOutput contains the result of umbrella chart generation.
type GeneratorOutput struct {
	// Files contains the paths of generated files.
	Files []string

	// TotalSize is the total size of all generated files.
	TotalSize int64

	// Duration is the time taken to generate the chart.
	Duration time.Duration

	// DeploymentSteps contains ordered deployment instructions for the user.
	DeploymentSteps []string
}

// Generator creates Helm umbrella charts from recipe results.
type Generator struct{}

// NewGenerator creates a new umbrella chart generator.
func NewGenerator() *Generator {
	return &Generator{}
}

// Generate creates an umbrella chart from the given input.
func (g *Generator) Generate(ctx context.Context, input *GeneratorInput, outputDir string) (*GeneratorOutput, error) {
	start := time.Now()

	output := &GeneratorOutput{
		Files: make([]string, 0),
	}

	if input == nil || input.RecipeResult == nil {
		return nil, errors.New(errors.ErrCodeInvalidRequest, "input and recipe result are required")
	}

	// Create output directory
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return nil, errors.Wrap(errors.ErrCodeInternal,
			"failed to create output directory", err)
	}

	// Generate Chart.yaml
	chartPath, chartSize, err := g.generateChartYAML(ctx, input, outputDir)
	if err != nil {
		return nil, errors.Wrap(errors.ErrCodeInternal,
			"failed to generate Chart.yaml", err)
	}
	output.Files = append(output.Files, chartPath)
	output.TotalSize += chartSize

	// Generate values.yaml
	valuesPath, valuesSize, err := g.generateValuesYAML(ctx, input, outputDir)
	if err != nil {
		return nil, errors.Wrap(errors.ErrCodeInternal,
			"failed to generate values.yaml", err)
	}
	output.Files = append(output.Files, valuesPath)
	output.TotalSize += valuesSize

	// Generate README.md
	readmePath, readmeSize, err := g.generateREADME(ctx, input, outputDir)
	if err != nil {
		return nil, errors.Wrap(errors.ErrCodeInternal,
			"failed to generate README.md", err)
	}
	output.Files = append(output.Files, readmePath)
	output.TotalSize += readmeSize

	// Generate checksums.txt if requested
	if input.IncludeChecksums {
		if err := checksum.GenerateChecksums(ctx, outputDir, output.Files); err != nil {
			return nil, errors.Wrap(errors.ErrCodeInternal,
				"failed to generate checksums", err)
		}
		checksumPath := checksum.GetChecksumFilePath(outputDir)
		info, statErr := os.Stat(checksumPath)
		if statErr == nil {
			output.Files = append(output.Files, checksumPath)
			output.TotalSize += info.Size()
		}
	}

	output.Duration = time.Since(start)

	// Populate deployment steps for CLI output
	output.DeploymentSteps = []string{
		fmt.Sprintf("cd %s", outputDir),
		"helm dependency update",
		"helm install cns-stack .",
	}

	slog.Debug("umbrella chart generated",
		"files", len(output.Files),
		"total_size", output.TotalSize,
		"duration", output.Duration,
	)

	return output, nil
}

// generateChartYAML creates the Chart.yaml file with dependencies.
func (g *Generator) generateChartYAML(ctx context.Context, input *GeneratorInput, outputDir string) (string, int64, error) {
	if err := ctx.Err(); err != nil {
		return "", 0, err
	}

	// Build dependencies from component refs in deployment order
	deps := make([]Dependency, 0, len(input.RecipeResult.ComponentRefs))

	// Create a map for quick lookup
	componentMap := make(map[string]recipe.ComponentRef)
	for _, ref := range input.RecipeResult.ComponentRefs {
		componentMap[ref.Name] = ref
	}

	// Add dependencies in deployment order
	for _, name := range input.RecipeResult.DeploymentOrder {
		ref, ok := componentMap[name]
		if !ok {
			continue
		}
		dep := Dependency{
			Name:       ref.Name,
			Version:    ref.Version,
			Repository: ref.Source,
		}
		// Add condition for optional enabling/disabling
		dep.Condition = fmt.Sprintf("%s.enabled", ref.Name)
		deps = append(deps, dep)
	}

	// Add any components not in deployment order (shouldn't happen, but be safe)
	for _, ref := range input.RecipeResult.ComponentRefs {
		found := false
		for _, d := range deps {
			if d.Name == ref.Name {
				found = true
				break
			}
		}
		if !found {
			deps = append(deps, Dependency{
				Name:       ref.Name,
				Version:    ref.Version,
				Repository: ref.Source,
				Condition:  fmt.Sprintf("%s.enabled", ref.Name),
			})
		}
	}

	// Build chart metadata
	chartName := "cns-stack"
	if input.RecipeResult.Criteria != nil {
		// Create a more descriptive name based on criteria
		parts := []string{"cns"}
		if input.RecipeResult.Criteria.Service != "" && input.RecipeResult.Criteria.Service != criteriaAny {
			parts = append(parts, string(input.RecipeResult.Criteria.Service))
		}
		if input.RecipeResult.Criteria.Accelerator != "" && input.RecipeResult.Criteria.Accelerator != criteriaAny {
			parts = append(parts, string(input.RecipeResult.Criteria.Accelerator))
		}
		if len(parts) > 1 {
			chartName = strings.Join(parts, "-")
		}
	}

	data := struct {
		ChartName    string
		Description  string
		Version      string
		AppVersion   string
		Dependencies []Dependency
	}{
		ChartName:    chartName,
		Description:  "NVIDIA Cloud Native Stack - GPU-accelerated Kubernetes deployment",
		Version:      normalizeVersion(input.Version),
		AppVersion:   input.RecipeResult.Metadata.Version,
		Dependencies: deps,
	}

	// Render template
	tmpl, err := template.New("Chart.yaml").Parse(chartTemplate)
	if err != nil {
		return "", 0, fmt.Errorf("failed to parse Chart.yaml template: %w", err)
	}

	var buf strings.Builder
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", 0, fmt.Errorf("failed to render Chart.yaml: %w", err)
	}

	// Write file
	chartPath := filepath.Join(outputDir, "Chart.yaml")
	content := buf.String()

	if err := os.WriteFile(chartPath, []byte(content), 0600); err != nil {
		return "", 0, fmt.Errorf("failed to write Chart.yaml: %w", err)
	}

	return chartPath, int64(len(content)), nil
}

// generateValuesYAML creates the values.yaml file with all component values.
func (g *Generator) generateValuesYAML(ctx context.Context, input *GeneratorInput, outputDir string) (string, int64, error) {
	if err := ctx.Err(); err != nil {
		return "", 0, err
	}

	// Build combined values map
	// Structure: component-name -> values
	values := make(map[string]interface{})

	// Add components in deployment order for consistent output
	for _, name := range input.RecipeResult.DeploymentOrder {
		if componentValues, ok := input.ComponentValues[name]; ok {
			// Add enabled flag (default true)
			componentWithEnabled := make(map[string]interface{})
			componentWithEnabled["enabled"] = true
			for k, v := range componentValues {
				componentWithEnabled[k] = v
			}
			values[name] = componentWithEnabled
		}
	}

	// Add any components not in deployment order
	for name, componentValues := range input.ComponentValues {
		if _, exists := values[name]; !exists {
			componentWithEnabled := make(map[string]interface{})
			componentWithEnabled["enabled"] = true
			for k, v := range componentValues {
				componentWithEnabled[k] = v
			}
			values[name] = componentWithEnabled
		}
	}

	// Generate YAML with header comment
	header := fmt.Sprintf(`# Cloud Native Stack - Helm Umbrella Chart Values
# Recipe Version: %s
# Bundler Version: %s
#
# This file contains configuration for all sub-charts.
# Each top-level key corresponds to a dependency in Chart.yaml.
# Set <component>.enabled=false to skip installing a component.
`, input.RecipeResult.Metadata.Version, input.Version)

	yamlBytes, err := yaml.Marshal(values)
	if err != nil {
		return "", 0, fmt.Errorf("failed to marshal values: %w", err)
	}

	content := header + string(yamlBytes)

	// Write file
	valuesPath := filepath.Join(outputDir, "values.yaml")
	if err := os.WriteFile(valuesPath, []byte(content), 0600); err != nil {
		return "", 0, fmt.Errorf("failed to write values.yaml: %w", err)
	}

	return valuesPath, int64(len(content)), nil
}

// generateREADME creates the README.md file with deployment instructions.
func (g *Generator) generateREADME(ctx context.Context, input *GeneratorInput, outputDir string) (string, int64, error) {
	if err := ctx.Err(); err != nil {
		return "", 0, err
	}

	// Build component list for README
	type ComponentInfo struct {
		Name       string
		Version    string
		Repository string
	}

	componentMap := make(map[string]recipe.ComponentRef)
	for _, ref := range input.RecipeResult.ComponentRefs {
		componentMap[ref.Name] = ref
	}

	components := make([]ComponentInfo, 0, len(input.RecipeResult.DeploymentOrder))
	for _, name := range input.RecipeResult.DeploymentOrder {
		if ref, ok := componentMap[name]; ok {
			components = append(components, ComponentInfo{
				Name:       ref.Name,
				Version:    ref.Version,
				Repository: ref.Source,
			})
		}
	}

	// Build criteria string for README
	criteriaLines := []string{}
	if input.RecipeResult.Criteria != nil {
		c := input.RecipeResult.Criteria
		if c.Service != "" && c.Service != criteriaAny {
			criteriaLines = append(criteriaLines, fmt.Sprintf("- **Service**: %s", c.Service))
		}
		if c.Accelerator != "" && c.Accelerator != criteriaAny {
			criteriaLines = append(criteriaLines, fmt.Sprintf("- **Accelerator**: %s", c.Accelerator))
		}
		if c.Intent != "" && c.Intent != criteriaAny {
			criteriaLines = append(criteriaLines, fmt.Sprintf("- **Intent**: %s", c.Intent))
		}
		if c.OS != "" && c.OS != criteriaAny {
			criteriaLines = append(criteriaLines, fmt.Sprintf("- **OS**: %s", c.OS))
		}
	}

	// Build constraints for README
	constraints := input.RecipeResult.Constraints

	data := struct {
		RecipeVersion  string
		BundlerVersion string
		Components     []ComponentInfo
		Criteria       []string
		Constraints    []recipe.Constraint
		ChartName      string
	}{
		RecipeVersion:  input.RecipeResult.Metadata.Version,
		BundlerVersion: input.Version,
		Components:     components,
		Criteria:       criteriaLines,
		Constraints:    constraints,
		ChartName:      "cns-stack",
	}

	// Render template
	tmpl, err := template.New("README.md").Parse(readmeTemplate)
	if err != nil {
		return "", 0, fmt.Errorf("failed to parse README.md template: %w", err)
	}

	var buf strings.Builder
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", 0, fmt.Errorf("failed to render README.md: %w", err)
	}

	// Write file
	readmePath := filepath.Join(outputDir, "README.md")
	content := buf.String()

	if err := os.WriteFile(readmePath, []byte(content), 0600); err != nil {
		return "", 0, fmt.Errorf("failed to write README.md: %w", err)
	}

	return readmePath, int64(len(content)), nil
}

// normalizeVersion ensures version string is valid for Helm (semver without 'v' prefix for chart version)
func normalizeVersion(v string) string {
	// Remove 'v' prefix if present for chart version
	v = strings.TrimPrefix(v, "v")
	// Default to 0.1.0 if empty
	if v == "" {
		return "0.1.0"
	}
	return v
}

// SortComponentsByDeploymentOrder sorts component names according to deployment order.
func SortComponentsByDeploymentOrder(components []string, deploymentOrder []string) []string {
	orderMap := make(map[string]int)
	for i, name := range deploymentOrder {
		orderMap[name] = i
	}

	sorted := make([]string, len(components))
	copy(sorted, components)

	sort.Slice(sorted, func(i, j int) bool {
		orderI, okI := orderMap[sorted[i]]
		orderJ, okJ := orderMap[sorted[j]]
		if okI && okJ {
			return orderI < orderJ
		}
		if okI {
			return true
		}
		if okJ {
			return false
		}
		return sorted[i] < sorted[j]
	})

	return sorted
}
