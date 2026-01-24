/*
Copyright © 2025 NVIDIA Corporation
SPDX-License-Identifier: Apache-2.0
*/

package bundler

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
	corev1 "k8s.io/api/core/v1"

	"github.com/NVIDIA/cloud-native-stack/pkg/bundler/config"
	"github.com/NVIDIA/cloud-native-stack/pkg/bundler/deployer/argocd"
	"github.com/NVIDIA/cloud-native-stack/pkg/bundler/deployer/helm"
	"github.com/NVIDIA/cloud-native-stack/pkg/bundler/result"
	"github.com/NVIDIA/cloud-native-stack/pkg/errors"
	"github.com/NVIDIA/cloud-native-stack/pkg/recipe"
)

// DefaultBundler generates Helm umbrella charts from recipes.
//
// The umbrella chart approach produces a single Helm chart with dependencies
// that can be deployed using standard Helm commands:
//
//	helm dependency update
//	helm install cns-stack . -f values.yaml
//
// Thread-safety: DefaultBundler is safe for concurrent use.
type DefaultBundler struct {
	// Config provides bundler-specific configuration including value overrides.
	Config *config.Config
}

// Option defines a functional option for configuring DefaultBundler.
type Option func(*DefaultBundler)

// WithConfig sets the bundler configuration.
// The config contains value overrides, node selectors, tolerations, etc.
func WithConfig(cfg *config.Config) Option {
	return func(db *DefaultBundler) {
		if cfg != nil {
			db.Config = cfg
		}
	}
}

// New creates a new DefaultBundler with the given options.
//
// Example:
//
//	b, err := bundler.New(
//	    bundler.WithConfig(config.NewConfig(
//	        config.WithValueOverrides(overrides),
//	    )),
//	)
func New(opts ...Option) (*DefaultBundler, error) {
	db := &DefaultBundler{
		Config: config.NewConfig(),
	}

	for _, opt := range opts {
		opt(db)
	}

	return db, nil
}

// NewWithConfig creates a new DefaultBundler with the given config.
// This is a convenience function equivalent to New(WithConfig(cfg)).
func NewWithConfig(cfg *config.Config) (*DefaultBundler, error) {
	return New(WithConfig(cfg))
}

// Make generates a deployment bundle from the given recipe.
// By default, generates a Helm umbrella chart. If deployer is set to "argocd",
// generates ArgoCD Application manifests.
//
// For umbrella chart output:
//   - Chart.yaml: Helm chart metadata with dependencies
//   - values.yaml: Combined values for all components
//   - README.md: Deployment instructions
//   - recipe.yaml: Copy of the input recipe
//   - checksums.txt: SHA256 checksums of generated files
//
// For ArgoCD output:
//   - app-of-apps.yaml: Parent ArgoCD Application
//   - <component>/application.yaml: ArgoCD Application per component
//   - <component>/values.yaml: Values for each component
//   - README.md: Deployment instructions
//
// Returns a result.Output summarizing the generation results.
func (b *DefaultBundler) Make(ctx context.Context, input recipe.RecipeInput, dir string) (*result.Output, error) {
	start := time.Now()

	// Validate input
	if input == nil {
		return nil, errors.New(errors.ErrCodeInvalidRequest, "recipe input cannot be nil")
	}

	// Only support RecipeResult format (not legacy Recipe)
	recipeResult, ok := input.(*recipe.RecipeResult)
	if !ok {
		return nil, errors.New(errors.ErrCodeInvalidRequest,
			"bundle generation requires RecipeResult format")
	}

	if len(recipeResult.ComponentRefs) == 0 {
		return nil, errors.New(errors.ErrCodeInvalidRequest,
			"recipe must contain at least one component reference")
	}

	// Set default output directory
	if dir == "" {
		dir = "."
	}

	// Create output directory
	if dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, errors.Wrap(errors.ErrCodeInternal,
				"failed to create output directory", err)
		}
	}

	// Extract values for each component from the recipe
	componentValues, err := b.extractComponentValues(ctx, recipeResult)
	if err != nil {
		return nil, errors.Wrap(errors.ErrCodeInternal,
			"failed to extract component values", err)
	}

	// Route based on deployer
	deployer := b.Config.Deployer()
	if deployer == config.DeployerArgoCD {
		return b.makeArgoCD(ctx, recipeResult, componentValues, dir, start)
	}
	return b.makeUmbrellaChart(ctx, recipeResult, componentValues, dir, start)
}

// makeUmbrellaChart generates a Helm umbrella chart.
func (b *DefaultBundler) makeUmbrellaChart(ctx context.Context, recipeResult *recipe.RecipeResult, componentValues map[string]map[string]interface{}, dir string, start time.Time) (*result.Output, error) {
	slog.Debug("generating umbrella chart",
		"component_count", len(recipeResult.ComponentRefs),
		"output_dir", dir,
	)

	// Generate umbrella chart
	generator := helm.NewGenerator()
	generatorInput := &helm.GeneratorInput{
		RecipeResult:     recipeResult,
		ComponentValues:  componentValues,
		Version:          b.Config.Version(),
		IncludeChecksums: b.Config.IncludeChecksums(),
	}

	output, err := generator.Generate(ctx, generatorInput, dir)
	if err != nil {
		return nil, errors.Wrap(errors.ErrCodeInternal,
			"failed to generate umbrella chart", err)
	}

	// Write recipe file
	recipeSize, err := b.writeRecipeFile(recipeResult, dir)
	if err != nil {
		return nil, errors.Wrap(errors.ErrCodeInternal,
			"failed to write recipe file", err)
	}

	// Build result output - includes umbrella chart files + recipe.yaml
	resultOutput := &result.Output{
		Results:       make([]*result.Result, 0),
		Errors:        make([]result.BundleError, 0),
		TotalDuration: time.Since(start),
		TotalSize:     output.TotalSize + recipeSize,
		TotalFiles:    len(output.Files) + 1, // +1 for recipe.yaml
		OutputDir:     dir,
	}

	// Add a single result for the umbrella chart
	umbrellaResult := &result.Result{
		Type:     "umbrella-chart",
		Success:  true,
		Files:    output.Files,
		Size:     output.TotalSize,
		Duration: output.Duration,
	}
	resultOutput.Results = append(resultOutput.Results, umbrellaResult)

	// Populate deployment info from generator output
	resultOutput.Deployment = &result.DeploymentInfo{
		Type:  "Helm umbrella chart",
		Steps: output.DeploymentSteps,
	}

	slog.Debug("umbrella chart generation complete",
		"files", len(output.Files),
		"size_bytes", output.TotalSize,
		"duration", output.Duration,
	)

	return resultOutput, nil
}

// makeArgoCD generates ArgoCD Application manifests.
func (b *DefaultBundler) makeArgoCD(ctx context.Context, recipeResult *recipe.RecipeResult, componentValues map[string]map[string]interface{}, dir string, start time.Time) (*result.Output, error) {
	slog.Debug("generating argocd applications",
		"component_count", len(recipeResult.ComponentRefs),
		"output_dir", dir,
	)

	// Generate ArgoCD applications
	generator := argocd.NewGenerator()
	generatorInput := &argocd.GeneratorInput{
		RecipeResult:     recipeResult,
		ComponentValues:  componentValues,
		Version:          b.Config.Version(),
		RepoURL:          b.Config.RepoURL(),
		IncludeChecksums: b.Config.IncludeChecksums(),
	}

	output, err := generator.Generate(ctx, generatorInput, dir)
	if err != nil {
		return nil, errors.Wrap(errors.ErrCodeInternal,
			"failed to generate argocd applications", err)
	}

	// Build result output
	resultOutput := &result.Output{
		Results:       make([]*result.Result, 0),
		Errors:        make([]result.BundleError, 0),
		TotalDuration: time.Since(start),
		TotalSize:     output.TotalSize,
		TotalFiles:    len(output.Files),
		OutputDir:     dir,
	}

	// Add a single result for the ArgoCD applications
	argocdResult := &result.Result{
		Type:     "argocd-applications",
		Success:  true,
		Files:    output.Files,
		Size:     output.TotalSize,
		Duration: output.Duration,
	}
	resultOutput.Results = append(resultOutput.Results, argocdResult)

	// Populate deployment info from generator output
	resultOutput.Deployment = &result.DeploymentInfo{
		Type:  "ArgoCD applications",
		Steps: output.DeploymentSteps,
		Notes: output.DeploymentNotes,
	}

	slog.Debug("argocd applications generation complete",
		"files", len(output.Files),
		"size_bytes", output.TotalSize,
		"duration", output.Duration,
	)

	return resultOutput, nil
}

// extractComponentValues extracts and processes values for each component in the recipe.
// It loads base values from the recipe, applies user overrides, and applies node selectors.
func (b *DefaultBundler) extractComponentValues(ctx context.Context, recipeResult *recipe.RecipeResult) (map[string]map[string]interface{}, error) {
	componentValues := make(map[string]map[string]interface{})

	for _, ref := range recipeResult.ComponentRefs {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		// Get base values from recipe
		values, err := recipeResult.GetValuesForComponent(ref.Name)
		if err != nil {
			slog.Warn("failed to get values for component, using empty map",
				"component", ref.Name,
				"error", err,
			)
			values = make(map[string]interface{})
		}

		// Apply user value overrides from --set flags
		if overrides := b.getValueOverridesForComponent(ref.Name); len(overrides) > 0 {
			if applyErr := applyMapOverrides(values, overrides); applyErr != nil {
				slog.Warn("failed to apply some value overrides",
					"component", ref.Name,
					"error", applyErr,
				)
			}
		}

		// Apply node selectors and tolerations based on component type
		b.applyNodeSchedulingOverrides(ref.Name, values)

		componentValues[ref.Name] = values
	}

	return componentValues, nil
}

// getValueOverridesForComponent returns value overrides for a specific component.
// Checks for both hyphenated (gpu-operator) and non-hyphenated (gpuoperator) keys.
func (b *DefaultBundler) getValueOverridesForComponent(componentName string) map[string]string {
	if b.Config == nil {
		return nil
	}

	allOverrides := b.Config.ValueOverrides()
	if allOverrides == nil {
		return nil
	}

	// Check exact name
	if overrides, ok := allOverrides[componentName]; ok {
		return overrides
	}

	// Check non-hyphenated version (e.g., gpuoperator for gpu-operator)
	nonHyphenated := removeHyphens(componentName)
	if nonHyphenated != componentName {
		if overrides, ok := allOverrides[nonHyphenated]; ok {
			return overrides
		}
	}

	return nil
}

// applyNodeSchedulingOverrides applies node selectors and tolerations to component values.
// Different components use different paths for these settings.
func (b *DefaultBundler) applyNodeSchedulingOverrides(componentName string, values map[string]interface{}) {
	if b.Config == nil {
		return
	}

	// Define component-specific paths for node scheduling
	type schedulingPaths struct {
		systemNodeSelector      []string
		systemTolerations       []string
		acceleratedNodeSelector []string
		acceleratedTolerations  []string
	}

	componentPaths := map[string]schedulingPaths{
		"gpu-operator": {
			systemNodeSelector:      []string{"operator.nodeSelector", "node-feature-discovery.gc.nodeSelector", "node-feature-discovery.master.nodeSelector"},
			systemTolerations:       []string{"operator.tolerations", "node-feature-discovery.gc.tolerations", "node-feature-discovery.master.tolerations"},
			acceleratedNodeSelector: []string{"daemonsets.nodeSelector", "node-feature-discovery.worker.nodeSelector"},
			acceleratedTolerations:  []string{"daemonsets.tolerations", "node-feature-discovery.worker.tolerations"},
		},
		"network-operator": {
			systemNodeSelector:      []string{"operator.nodeSelector"},
			acceleratedNodeSelector: []string{"daemonsets.nodeSelector"},
		},
		"cert-manager": {
			systemNodeSelector: []string{"controller.nodeSelector"},
			systemTolerations:  []string{"controller.tolerations"},
		},
	}

	paths, ok := componentPaths[componentName]
	if !ok {
		return // Unknown component, skip
	}

	// Apply system node selector
	if nodeSelector := b.Config.SystemNodeSelector(); len(nodeSelector) > 0 && len(paths.systemNodeSelector) > 0 {
		applyNodeSelectorOverrides(values, nodeSelector, paths.systemNodeSelector...)
	}

	// Apply system tolerations
	if tolerations := b.Config.SystemNodeTolerations(); len(tolerations) > 0 && len(paths.systemTolerations) > 0 {
		applyTolerationsOverrides(values, tolerations, paths.systemTolerations...)
	}

	// Apply accelerated node selector
	if nodeSelector := b.Config.AcceleratedNodeSelector(); len(nodeSelector) > 0 && len(paths.acceleratedNodeSelector) > 0 {
		applyNodeSelectorOverrides(values, nodeSelector, paths.acceleratedNodeSelector...)
	}

	// Apply accelerated tolerations
	if tolerations := b.Config.AcceleratedNodeTolerations(); len(tolerations) > 0 && len(paths.acceleratedTolerations) > 0 {
		applyTolerationsOverrides(values, tolerations, paths.acceleratedTolerations...)
	}
}

// writeRecipeFile serializes the recipe to the bundle directory.
func (b *DefaultBundler) writeRecipeFile(recipeResult *recipe.RecipeResult, dir string) (int64, error) {
	recipeData, err := yaml.Marshal(recipeResult)
	if err != nil {
		return 0, fmt.Errorf("failed to serialize recipe: %w", err)
	}

	recipePath := fmt.Sprintf("%s/recipe.yaml", dir)
	if err := os.WriteFile(recipePath, recipeData, 0600); err != nil {
		return 0, fmt.Errorf("failed to write recipe file: %w", err)
	}

	slog.Debug("wrote recipe file", "path", recipePath)
	return int64(len(recipeData)), nil
}

// removeHyphens removes hyphens from a string.
func removeHyphens(s string) string {
	result := make([]byte, 0, len(s))
	for i := 0; i < len(s); i++ {
		if s[i] != '-' {
			result = append(result, s[i])
		}
	}
	return string(result)
}

// applyMapOverrides applies overrides to a map[string]interface{} using dot-notation paths.
// Handles nested maps by traversing the path segments and creating nested maps as needed.
func applyMapOverrides(target map[string]interface{}, overrides map[string]string) error {
	if target == nil {
		return fmt.Errorf("target map cannot be nil")
	}

	if len(overrides) == 0 {
		return nil
	}

	var errs []string
	for path, value := range overrides {
		if err := setMapValueByPath(target, path, value); err != nil {
			errs = append(errs, fmt.Sprintf("%s=%s: %v", path, value, err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("failed to apply map overrides: %s", strings.Join(errs, "; "))
	}

	return nil
}

// setMapValueByPath sets a value in a nested map using dot-notation path.
// Creates nested maps as needed. Converts string values to bools/numbers when appropriate.
func setMapValueByPath(target map[string]interface{}, path, value string) error {
	parts := strings.Split(path, ".")
	current := target

	// Traverse/create the path up to the last segment
	for i := 0; i < len(parts)-1; i++ {
		part := parts[i]
		if next, ok := current[part]; ok {
			// If the value exists, it must be a map
			if nextMap, ok := next.(map[string]interface{}); ok {
				current = nextMap
			} else {
				return fmt.Errorf("path segment %q exists but is not a map (type: %T)", part, next)
			}
		} else {
			// Create a new nested map
			newMap := make(map[string]interface{})
			current[part] = newMap
			current = newMap
		}
	}

	// Set the final value
	lastPart := parts[len(parts)-1]
	current[lastPart] = convertMapValue(value)

	return nil
}

// convertMapValue converts a string value to an appropriate Go type.
// Handles bools ("true"/"false") and numbers.
func convertMapValue(value string) interface{} {
	// Try bool conversion
	if value == "true" {
		return true
	}
	if value == "false" {
		return false
	}

	// Try integer conversion
	if i, err := strconv.ParseInt(value, 10, 64); err == nil {
		return i
	}

	// Try float conversion
	if f, err := strconv.ParseFloat(value, 64); err == nil {
		return f
	}

	// Return as string
	return value
}

// applyNodeSelectorOverrides applies node selector values to the specified paths in a values map.
func applyNodeSelectorOverrides(values map[string]interface{}, nodeSelector map[string]string, paths ...string) {
	if len(nodeSelector) == 0 || len(paths) == 0 {
		return
	}

	for _, path := range paths {
		// Convert node selector to interface map for YAML compatibility
		selectorMap := make(map[string]interface{})
		for k, v := range nodeSelector {
			selectorMap[k] = v
		}
		_ = setMapValueByPath(values, path, "")
		setNestedMapValue(values, path, selectorMap)
	}
}

// applyTolerationsOverrides applies tolerations to the specified paths in a values map.
func applyTolerationsOverrides(values map[string]interface{}, tolerations []corev1.Toleration, paths ...string) {
	if len(tolerations) == 0 || len(paths) == 0 {
		return
	}

	// Convert tolerations to interface slice
	tolerationsList := make([]interface{}, 0, len(tolerations))
	for _, t := range tolerations {
		tolMap := make(map[string]interface{})
		if t.Key != "" {
			tolMap["key"] = t.Key
		}
		if t.Operator != "" {
			tolMap["operator"] = string(t.Operator)
		}
		if t.Value != "" {
			tolMap["value"] = t.Value
		}
		if t.Effect != "" {
			tolMap["effect"] = string(t.Effect)
		}
		if t.TolerationSeconds != nil {
			tolMap["tolerationSeconds"] = *t.TolerationSeconds
		}
		tolerationsList = append(tolerationsList, tolMap)
	}

	for _, path := range paths {
		setNestedMapValue(values, path, tolerationsList)
	}
}

// setNestedMapValue sets a value at a nested path in a map.
func setNestedMapValue(target map[string]interface{}, path string, value interface{}) {
	parts := strings.Split(path, ".")
	current := target

	// Traverse/create the path up to the last segment
	for i := 0; i < len(parts)-1; i++ {
		part := parts[i]
		if next, ok := current[part]; ok {
			if nextMap, ok := next.(map[string]interface{}); ok {
				current = nextMap
			} else {
				// Existing value is not a map, create new map
				newMap := make(map[string]interface{})
				current[part] = newMap
				current = newMap
			}
		} else {
			newMap := make(map[string]interface{})
			current[part] = newMap
			current = newMap
		}
	}

	// Set the final value
	lastPart := parts[len(parts)-1]
	current[lastPart] = value
}
