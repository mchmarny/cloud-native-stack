/*
Copyright © 2025 NVIDIA Corporation
SPDX-License-Identifier: Apache-2.0
*/
package cli

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/NVIDIA/cloud-native-stack/pkg/bundler"
	"github.com/NVIDIA/cloud-native-stack/pkg/bundler/config"
	"github.com/NVIDIA/cloud-native-stack/pkg/bundler/registry"
	"github.com/NVIDIA/cloud-native-stack/pkg/bundler/result"
	"github.com/NVIDIA/cloud-native-stack/pkg/bundler/types"
	deployerRegistry "github.com/NVIDIA/cloud-native-stack/pkg/deployer/registry"
	deployerTypes "github.com/NVIDIA/cloud-native-stack/pkg/deployer/types"
	"github.com/NVIDIA/cloud-native-stack/pkg/oci"
	"github.com/NVIDIA/cloud-native-stack/pkg/recipe"
	"github.com/NVIDIA/cloud-native-stack/pkg/serializer"
	"github.com/NVIDIA/cloud-native-stack/pkg/snapshotter"
	"github.com/urfave/cli/v3"
)

// Output format constants.
const (
	outputFormatDir = "dir"
	outputFormatOCI = "oci"
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
				Aliases:  []string{"f"},
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
			// OCI push flags
			&cli.BoolFlag{
				Name:  "push",
				Usage: "Push generated bundle as OCI artifact to registry",
			// Output format flags
			&cli.StringFlag{
				Name:    "output-format",
				Aliases: []string{"F"},
				Value:   outputFormatDir,
				Usage:   "Output format: dir (local directory) or oci (push to OCI registry)",
			},
			// OCI registry flags (only used when output-format is oci)
			&cli.StringFlag{
				Name:  "registry",
				Usage: "OCI registry host (e.g., ghcr.io, localhost:5000)",
			},
			&cli.StringFlag{
				Name:  "repository",
				Usage: "OCI repository path (e.g., nvidia/eidos)",
			},
			&cli.StringFlag{
				Name:  "tag",
				Usage: "OCI image tag (default: latest)",
			},
			&cli.BoolFlag{
				Name:  "insecure-tls",
				Usage: "Skip TLS certificate verification for registry",
			},
			&cli.BoolFlag{
				Name:  "plain-http",
				Usage: "Use HTTP instead of HTTPS for registry connection (for local development)",
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			recipeFilePath := cmd.String("recipe")
			outputDir := cmd.String("output")
			bundlerTypesStr := cmd.StringSlice("bundlers")
			setFlags := cmd.StringSlice("set")

			// Output format and OCI options
			outputFormat := cmd.String("output-format")
			registryHost := cmd.String("registry")
			repository := cmd.String("repository")
			tag := cmd.String("tag")
			insecureTLS := cmd.Bool("insecure-tls")
			plainHTTP := cmd.Bool("plain-http")

			// Validate output-format
			if outputFormat != outputFormatDir && outputFormat != outputFormatOCI {
				return fmt.Errorf("--output-format must be '%s' or '%s', got '%s'", outputFormatDir, outputFormatOCI, outputFormat)
			}

			// Validate OCI flags when output-format is oci
			if outputFormat == outputFormatOCI {
				if registryHost == "" {
					return fmt.Errorf("--registry is required when --output-format is 'oci'")
				}
				if repository == "" {
					return fmt.Errorf("--repository is required when --output-format is 'oci'")
				}
			}

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

			// Determine output directory - use temp dir for OCI mode
			var bundleOutputDir string
			var cleanupTempDir func()

			if outputFormat == outputFormatOCI {
				// Create temp directory for OCI output
				tempDir, tempErr := os.MkdirTemp("", "eidos-bundle-*")
				if tempErr != nil {
					return fmt.Errorf("failed to create temp directory: %w", tempErr)
				}
				bundleOutputDir = tempDir
				cleanupTempDir = func() { os.RemoveAll(tempDir) }
			} else {
				bundleOutputDir = outputDir
			}

			out, err := b.Make(ctx, rec, bundleOutputDir)
			if err != nil {
				if cleanupTempDir != nil {
					cleanupTempDir()
				}
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
				if cleanupTempDir != nil {
					cleanupTempDir()
				}
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

			// Push to OCI registry if output-format is oci
			if outputFormat == outputFormatOCI {
				if pushErr := pushToOCI(ctx, ociPushConfig{
					sourceDir:   bundleOutputDir,
					registry:    registryHost,
					repository:  repository,
					tag:         tag,
					plainHTTP:   plainHTTP,
					insecureTLS: insecureTLS,
				}, out.Results); pushErr != nil {
					if cleanupTempDir != nil {
						cleanupTempDir()
					}
					return pushErr
				}
			}

			// Cleanup temp dir if OCI mode
			if cleanupTempDir != nil {
				cleanupTempDir()
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
// ociPushConfig holds configuration for OCI push operations.
type ociPushConfig struct {
	sourceDir   string
	registry    string
	repository  string
	tag         string
	plainHTTP   bool
	insecureTLS bool
}

// pushToOCI pushes the bundle to an OCI registry and updates results with metadata.
func pushToOCI(ctx context.Context, cfg ociPushConfig, results []*result.Result) error {
	absOutputDir, err := filepath.Abs(cfg.sourceDir)
	if err != nil {
		return fmt.Errorf("failed to resolve output directory: %w", err)
	}

	// Default tag to "latest" if not provided
	imageTag := cfg.tag
	if imageTag == "" {
		imageTag = "latest"
	}

	slog.Info("pushing bundle to OCI registry",
		"registry", cfg.registry,
		"repository", cfg.repository,
		"tag", imageTag,
	)

	pushResult, err := oci.Push(ctx, oci.PushOptions{
		SourceDir:   absOutputDir,
		Registry:    cfg.registry,
		Repository:  cfg.repository,
		Tag:         imageTag,
		PlainHTTP:   cfg.plainHTTP,
		InsecureTLS: cfg.insecureTLS,
	})
	if err != nil {
		return fmt.Errorf("failed to push OCI artifact: %w", err)
	}

	// Update results with OCI metadata
	for i := range results {
		if results[i].Success {
			results[i].SetOCIMetadata(pushResult.Digest, pushResult.Reference)
		}
	}

	slog.Info("OCI artifact pushed successfully",
		"reference", pushResult.Reference,
		"digest", pushResult.Digest,
	)

	return nil
}
