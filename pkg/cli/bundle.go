/*
Copyright © 2025 NVIDIA Corporation
SPDX-License-Identifier: Apache-2.0
*/
package cli

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/NVIDIA/cloud-native-stack/pkg/bundler"
	"github.com/NVIDIA/cloud-native-stack/pkg/bundler/config"
	"github.com/NVIDIA/cloud-native-stack/pkg/bundler/registry"
	"github.com/NVIDIA/cloud-native-stack/pkg/bundler/types"
	deployerRegistry "github.com/NVIDIA/cloud-native-stack/pkg/deployer/registry"
	deployerTypes "github.com/NVIDIA/cloud-native-stack/pkg/deployer/types"
	"github.com/NVIDIA/cloud-native-stack/pkg/recipe"
	"github.com/NVIDIA/cloud-native-stack/pkg/serializer"
	"github.com/NVIDIA/cloud-native-stack/pkg/snapshotter"
	"github.com/urfave/cli/v3"
)

func bundleCmd() *cli.Command {
	return &cli.Command{
		Name:                  "bundle",
		EnableShellCompletion: true,
		Usage:                 "Generate artifact bundle from a given recipe",
		Description: `Generates a bundle of artifacts from a given recipe parameters including:
  - Kubernetes manifests
  - Helm charts
  - Installation scripts
  - Configuration files
`,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "recipe",
				Aliases:  []string{"r"},
				Required: true,
				Usage: `Path/URI to previously generated recipe from which to build the bundle. 
	Supports: file paths, HTTP/HTTPS URLs, or ConfigMap URIs (cm://namespace/name).`,
			},
			&cli.StringSliceFlag{
				Name:    "bundlers",
				Aliases: []string{"b"},
				Usage: fmt.Sprintf(`Types of bundlers to execute (supported: %s). 
	If not specified, all supported bundlers are executed.`,
					strings.Join(types.SupportedBundleTypesAsStrings(), ", ")),
			},
			&cli.StringFlag{
				Name:    "output",
				Aliases: []string{"o"},
				Value:   ".",
				Usage:   "output directory path",
			},
			&cli.StringSliceFlag{
				Name:  "set",
				Usage: "Override values in generated bundle files (format: bundler:path.to.field=value, e.g., --set gpuoperator:gds.enabled=true)",
			},
			&cli.StringSliceFlag{
				Name:  "system-node-selector",
				Usage: "Node selector for system components (format: key=value, can be repeated)",
			},
			&cli.StringSliceFlag{
				Name:  "system-node-toleration",
				Usage: "Toleration for system components (format: key=value:effect, can be repeated)",
			},
			&cli.StringSliceFlag{
				Name:  "accelerated-node-selector",
				Usage: "Node selector for accelerated/GPU nodes (format: key=value, can be repeated)",
			},
			&cli.StringSliceFlag{
				Name:  "accelerated-node-toleration",
				Usage: "Toleration for accelerated/GPU nodes (format: key=value:effect, can be repeated)",
			},
			&cli.StringFlag{
				Name:  "deployer",
				Value: string(deployerTypes.DeployerTypeScript),
				Usage: fmt.Sprintf("Deployment method for generated components (supported: %s)",
					strings.Join(deployerTypesToStrings(deployerTypes.AllDeployerTypes()), ", ")),
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			recipeFilePath := cmd.String("recipe")
			outputDir := cmd.String("output")
			bundlerTypesStr := cmd.StringSlice("bundlers")
			setFlags := cmd.StringSlice("set")

			// Parse value overrides from --set flags
			valueOverrides, err := parseSetFlags(setFlags)
			if err != nil {
				return fmt.Errorf("invalid --set flag: %w", err)
			}

			// Parse node selectors
			systemNodeSelector, err := snapshotter.ParseNodeSelectors(cmd.StringSlice("system-node-selector"))
			if err != nil {
				return fmt.Errorf("invalid --system-node-selector: %w", err)
			}
			acceleratedNodeSelector, err := snapshotter.ParseNodeSelectors(cmd.StringSlice("accelerated-node-selector"))
			if err != nil {
				return fmt.Errorf("invalid --accelerated-node-selector: %w", err)
			}

			// Parse tolerations
			systemNodeTolerations, err := snapshotter.ParseTolerations(cmd.StringSlice("system-node-toleration"))
			if err != nil {
				return fmt.Errorf("invalid --system-node-toleration: %w", err)
			}
			acceleratedNodeTolerations, err := snapshotter.ParseTolerations(cmd.StringSlice("accelerated-node-toleration"))
			if err != nil {
				return fmt.Errorf("invalid --accelerated-node-toleration: %w", err)
			}

			// Parse and validate deployer type
			deployerTypeStr := cmd.String("deployer")
			deployerType := deployerTypes.DeployerType(deployerTypeStr)
			if !deployerType.IsValid() {
				// Get list of valid types from registry
				deployerReg := deployerRegistry.NewFromGlobal()
				registeredTypes := deployerReg.Types()
				return fmt.Errorf("invalid deployer type '%s': must be one of %s",
					deployerTypeStr, strings.Join(deployerTypesToStrings(registeredTypes), ", "))
			}

			// Parse bundler types
			var bundlerTypes []types.BundleType
			for _, t := range bundlerTypesStr {
				bt, parseErr := types.ParseType(t)
				if parseErr != nil {
					return fmt.Errorf("invalid bundler type '%s': %w", t, parseErr)
				}
				bundlerTypes = append(bundlerTypes, bt)
			}
			slog.Info("generating bundle",
				slog.String("recipeFilePath", recipeFilePath),
				slog.String("outputDir", outputDir),
				slog.Any("bundlerTypes", bundlerTypes),
				slog.String("deployerType", string(deployerType)),
			)

			rec, err := serializer.FromFile[recipe.RecipeResult](recipeFilePath)
			if err != nil {
				slog.Error("failed to load recipe file", "error", err, "path", recipeFilePath)
				return err
			}

			// Create bundler registry from global with config
			reg := registry.NewFromGlobal(
				config.NewConfig(
					config.WithVersion(version),
					config.WithValueOverrides(valueOverrides),
					config.WithSystemNodeSelector(systemNodeSelector),
					config.WithSystemNodeTolerations(systemNodeTolerations),
					config.WithAcceleratedNodeSelector(acceleratedNodeSelector),
					config.WithAcceleratedNodeTolerations(acceleratedNodeTolerations),
				),
			)

			// Create bundler instance
			b, err := bundler.New(
				// If bundler types are not specified, all supported bundlers are used.
				// An empty or nil slice means all bundlers as well.
				bundler.WithBundlerTypes(bundlerTypes),
				bundler.WithRegistry(reg),
				bundler.WithDeployer(deployerType),
			)
			if err != nil {
				slog.Error("failed to create bundler", "error", err)
				return err
			}

			out, err := b.Make(ctx, rec, outputDir)
			if err != nil {
				slog.Error("bundle generation failed", "error", err)
				return err
			}

			slog.Info("bundle generation completed",
				"success", out.SuccessCount(),
				"errors", len(out.Errors),
				"duration_sec", out.TotalDuration.Seconds(),
				"summary", out.Summary(),
			)

			// Return error if any bundlers failed
			if out.HasErrors() {
				// Log each bundler error for debugging
				for _, bundleErr := range out.Errors {
					slog.Error("bundler failed",
						"bundler_type", bundleErr.BundlerType,
						"error", bundleErr.Error,
					)
				}
				return fmt.Errorf("bundle generation completed with errors: %d/%d bundlers failed",
					len(out.Errors), len(out.Results))
			}

			return nil
		},
	}
}

// parseSetFlags parses --set flags in format "bundler:path.to.field=value"
// Returns a map of bundler -> (path -> value)
func parseSetFlags(setFlags []string) (map[string]map[string]string, error) {
	overrides := make(map[string]map[string]string)

	for _, setFlag := range setFlags {
		// Split on first ':' to get bundler and path=value
		parts := strings.SplitN(setFlag, ":", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid format '%s': expected 'bundler:path=value'", setFlag)
		}

		bundlerName := parts[0]
		pathValue := parts[1]

		// Split on first '=' to get path and value
		kvParts := strings.SplitN(pathValue, "=", 2)
		if len(kvParts) != 2 {
			return nil, fmt.Errorf("invalid format '%s': expected 'bundler:path=value'", setFlag)
		}

		path := kvParts[0]
		value := kvParts[1]

		if path == "" || value == "" {
			return nil, fmt.Errorf("invalid format '%s': path and value cannot be empty", setFlag)
		}

		// Initialize bundler map if needed
		if overrides[bundlerName] == nil {
			overrides[bundlerName] = make(map[string]string)
		}

		overrides[bundlerName][path] = value
	}

	return overrides, nil
}

// deployerTypesToStrings converts deployer types to string slice for help text
func deployerTypesToStrings(types []deployerTypes.DeployerType) []string {
	result := make([]string, len(types))
	for i, t := range types {
		result[i] = string(t)
	}
	return result
}
