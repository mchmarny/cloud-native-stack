/*
Copyright © 2025 NVIDIA Corporation
SPDX-License-Identifier: Apache-2.0
*/
package cli

import (
	"context"
	"fmt"
	"log/slog"
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
			},
			// Output format flag
			&cli.StringFlag{
				Name:    "output-format",
				Aliases: []string{"F"},
				Value:   outputFormatDir,
				Usage:   "Output format: dir (local directory) or oci (OCI Image Layout)",
			},
			// OCI registry flags (used when output-format is oci)
			&cli.StringFlag{
				Name:  "registry",
				Usage: "OCI registry host for image reference (e.g., ghcr.io, localhost:5000)",
			},
			&cli.StringFlag{
				Name:  "repository",
				Usage: "OCI repository path for image reference (e.g., nvidia/eidos)",
			},
			&cli.StringFlag{
				Name:  "tag",
				Usage: "OCI image tag (default: latest)",
			},
			// Push flag - controls whether to push to remote registry
			&cli.BoolFlag{
				Name:  "push",
				Usage: "Push OCI artifact to remote registry (requires --output-format=oci)",
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
			push := cmd.Bool("push")
			insecureTLS := cmd.Bool("insecure-tls")
			plainHTTP := cmd.Bool("plain-http")

			// Validate output-format
			if outputFormat != outputFormatDir && outputFormat != outputFormatOCI {
				return fmt.Errorf("--output-format must be '%s' or '%s', got '%s'", outputFormatDir, outputFormatOCI, outputFormat)
			}

			// Validate --push requires --output-format=oci
			if push && outputFormat != outputFormatOCI {
				return fmt.Errorf("--push requires --output-format=oci")
			}

			// Validate OCI flags when output-format is oci
			if outputFormat == outputFormatOCI {
				if registryHost == "" {
					return fmt.Errorf("--registry is required when --output-format is 'oci'")
				}
				if repository == "" {
					return fmt.Errorf("--repository is required when --output-format is 'oci'")
				}
				// Validate registry and repository format early
				if err := oci.ValidateRegistryReference(registryHost, repository); err != nil {
					return err
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

			// Generate bundle
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

			// Package as OCI artifact when output-format is oci
			if outputFormat == outputFormatOCI {
				if ociErr := handleOCIOutput(ctx, ociConfig{
					sourceDir:   outputDir,
					outputDir:   outputDir,
					registry:    registryHost,
					repository:  repository,
					tag:         tag,
					push:        push,
					plainHTTP:   plainHTTP,
					insecureTLS: insecureTLS,
				}, out.Results); ociErr != nil {
					return ociErr
				}
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

// ociConfig holds configuration for OCI operations.
type ociConfig struct {
	sourceDir   string
	outputDir   string
	registry    string
	repository  string
	tag         string
	push        bool
	plainHTTP   bool
	insecureTLS bool
}

// handleOCIOutput packages the bundle as an OCI artifact and optionally pushes to a remote registry.
// When --push is specified, the artifact is also pushed to the remote registry.
// OCI metadata (digest, reference) is populated on results when --output-format=oci is used.
func handleOCIOutput(ctx context.Context, cfg ociConfig, results []*result.Result) error {
	absSourceDir, err := filepath.Abs(cfg.sourceDir)
	if err != nil {
		return fmt.Errorf("failed to resolve source directory: %w", err)
	}

	absOutputDir, err := filepath.Abs(cfg.outputDir)
	if err != nil {
		return fmt.Errorf("failed to resolve output directory: %w", err)
	}

	// Default tag to "latest" if not provided
	imageTag := cfg.tag
	if imageTag == "" {
		imageTag = "latest"
	}

	slog.Info("packaging bundle as OCI artifact",
		"registry", cfg.registry,
		"repository", cfg.repository,
		"tag", imageTag,
		"push", cfg.push,
	)

	// Package locally first
	packageResult, err := oci.Package(ctx, oci.PackageOptions{
		SourceDir:  absSourceDir,
		OutputDir:  absOutputDir,
		Registry:   cfg.registry,
		Repository: cfg.repository,
		Tag:        imageTag,
	})
	if err != nil {
		return fmt.Errorf("failed to package OCI artifact: %w", err)
	}

	slog.Info("OCI artifact packaged locally",
		"reference", packageResult.Reference,
		"digest", packageResult.Digest,
		"store_path", packageResult.StorePath,
	)

	// Update results with OCI metadata
	for i := range results {
		if results[i].Success {
			results[i].SetOCIMetadata(packageResult.Digest, packageResult.Reference)
		}
	}

	// Push to remote registry if requested
	if cfg.push {
		slog.Info("pushing OCI artifact to remote registry",
			"registry", cfg.registry,
			"repository", cfg.repository,
			"tag", imageTag,
		)

		pushResult, pushErr := oci.PushFromStore(ctx, packageResult.StorePath, oci.PushOptions{
			Registry:    cfg.registry,
			Repository:  cfg.repository,
			Tag:         imageTag,
			PlainHTTP:   cfg.plainHTTP,
			InsecureTLS: cfg.insecureTLS,
		})
		if pushErr != nil {
			return fmt.Errorf("failed to push OCI artifact to registry: %w", pushErr)
		}

		slog.Info("OCI artifact pushed successfully",
			"reference", pushResult.Reference,
			"digest", pushResult.Digest,
		)
	}

	return nil
}
