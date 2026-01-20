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

	corev1 "k8s.io/api/core/v1"

	"github.com/NVIDIA/cloud-native-stack/pkg/bundler"
	"github.com/NVIDIA/cloud-native-stack/pkg/bundler/config"
	"github.com/NVIDIA/cloud-native-stack/pkg/bundler/result"
	"github.com/NVIDIA/cloud-native-stack/pkg/oci"
	"github.com/NVIDIA/cloud-native-stack/pkg/recipe"
	"github.com/NVIDIA/cloud-native-stack/pkg/serializer"
	"github.com/NVIDIA/cloud-native-stack/pkg/snapshotter"
	"github.com/urfave/cli/v3"
)

// deployerArgoCD is the ArgoCD deployer type.
const deployerArgoCD = "argocd"

// Output format constants.
const (
	outputFormatDir = "dir"
	outputFormatOCI = "oci"
	defaultOCITag   = "latest"
)

// bundleCmdOptions holds parsed options for the bundle command.
type bundleCmdOptions struct {
	recipeFilePath             string
	outputDir                  string
	kubeconfig                 string
	deployer                   string
	repoURL                    string
	valueOverrides             map[string]map[string]string
	systemNodeSelector         map[string]string
	systemNodeTolerations      []corev1.Toleration
	acceleratedNodeSelector    map[string]string
	acceleratedNodeTolerations []corev1.Toleration
	outputFormat               string
	registryHost               string
	repository                 string
	tag                        string
	push                       bool
	plainHTTP                  bool
	insecureTLS                bool
}

// parseBundleCmdOptions parses and validates command options.
func parseBundleCmdOptions(cmd *cli.Command) (*bundleCmdOptions, error) {
	opts := &bundleCmdOptions{
		recipeFilePath: cmd.String("recipe"),
		outputDir:      cmd.String("output"),
		kubeconfig:     cmd.String("kubeconfig"),
		deployer:       cmd.String("deployer"),
		repoURL:        cmd.String("repo"),
		outputFormat:   cmd.String("output-format"),
		registryHost:   cmd.String("registry"),
		repository:     cmd.String("repository"),
		tag:            cmd.String("tag"),
		push:           cmd.Bool("push"),
		insecureTLS:    cmd.Bool("insecure-tls"),
		plainHTTP:      cmd.Bool("plain-http"),
	}

	// Validate deployer flag
	if opts.deployer != "" && opts.deployer != deployerArgoCD {
		return nil, fmt.Errorf("invalid --deployer value: %q (must be '' or 'argocd')", opts.deployer)
	}

	// Validate output-format
	if opts.outputFormat != outputFormatDir && opts.outputFormat != outputFormatOCI {
		return nil, fmt.Errorf("--output-format must be '%s' or '%s', got '%s'",
			outputFormatDir, outputFormatOCI, opts.outputFormat)
	}

	// Validate --push requires --output-format=oci
	if opts.push && opts.outputFormat != outputFormatOCI {
		return nil, fmt.Errorf("--push requires --output-format=oci")
	}

	// Validate OCI flags when output-format is oci
	if opts.outputFormat == outputFormatOCI {
		if opts.registryHost == "" {
			return nil, fmt.Errorf("--registry is required when --output-format is 'oci'")
		}
		if opts.repository == "" {
			return nil, fmt.Errorf("--repository is required when --output-format is 'oci'")
		}
		if err := oci.ValidateRegistryReference(opts.registryHost, opts.repository); err != nil {
			return nil, fmt.Errorf("invalid OCI reference: %w", err)
		}
	}

	// Parse value overrides from --set flags
	var err error
	opts.valueOverrides, err = config.ParseValueOverrides(cmd.StringSlice("set"))
	if err != nil {
		return nil, fmt.Errorf("invalid --set flag: %w", err)
	}

	// Parse node selectors
	opts.systemNodeSelector, err = snapshotter.ParseNodeSelectors(cmd.StringSlice("system-node-selector"))
	if err != nil {
		return nil, fmt.Errorf("invalid --system-node-selector: %w", err)
	}
	opts.acceleratedNodeSelector, err = snapshotter.ParseNodeSelectors(cmd.StringSlice("accelerated-node-selector"))
	if err != nil {
		return nil, fmt.Errorf("invalid --accelerated-node-selector: %w", err)
	}

	// Parse tolerations
	opts.systemNodeTolerations, err = snapshotter.ParseTolerations(cmd.StringSlice("system-node-toleration"))
	if err != nil {
		return nil, fmt.Errorf("invalid --system-node-toleration: %w", err)
	}
	opts.acceleratedNodeTolerations, err = snapshotter.ParseTolerations(cmd.StringSlice("accelerated-node-toleration"))
	if err != nil {
		return nil, fmt.Errorf("invalid --accelerated-node-toleration: %w", err)
	}

	return opts, nil
}

func bundleCmd() *cli.Command {
	return &cli.Command{
		Name:                  "bundle",
		EnableShellCompletion: true,
		Usage:                 "Generate deployment bundle from a given recipe",
		Description: `Generates a deployment bundle from a given recipe. By default, this creates
a Helm umbrella chart. Use --deployer argocd to generate ArgoCD Applications.

# Default Output (Helm Umbrella Chart)

  - Chart.yaml: Helm chart metadata with component dependencies
  - values.yaml: Combined values for all components
  - README.md: Deployment instructions
  - recipe.yaml: Copy of the input recipe for reference

# ArgoCD Output (--deployer argocd)

  - app-of-apps.yaml: Parent ArgoCD Application
  - <component>/application.yaml: ArgoCD Application per component
  - <component>/values.yaml: Values for each component
  - README.md: Deployment instructions

# Examples

Generate Helm umbrella chart (default):
  cnsctl bundle --recipe recipe.yaml --output ./my-bundle

Generate ArgoCD App of Apps:
  cnsctl bundle --recipe recipe.yaml --output ./my-bundle --deployer argocd

Override values in generated bundle:
  cnsctl bundle --recipe recipe.yaml --set gpuoperator:driver.version=570.133.20

Set node selectors for GPU workloads:
  cnsctl bundle --recipe recipe.yaml \
    --accelerated-node-selector nodeGroup=gpu-nodes \
    --accelerated-node-toleration nvidia.com/gpu=present:NoSchedule

Package bundle as OCI artifact (local only):
  cnsctl bundle --recipe recipe.yaml --output ./my-bundle --output-format oci \
    --registry ghcr.io --repository nvidia/cns-bundle --tag v1.0.0

Package and push bundle to OCI registry:
  cnsctl bundle --recipe recipe.yaml --output ./my-bundle --output-format oci \
    --registry ghcr.io --repository nvidia/cns-bundle --tag v1.0.0 --push

# Deployment (Helm)

After generating the Helm bundle, deploy using:
  cd my-bundle
  helm dependency update
  helm install cns-stack . -n cns-system --create-namespace

# Deployment (ArgoCD)

After generating the ArgoCD bundle:
  1. Push the generated files to your GitOps repository
  2. Apply the app-of-apps.yaml to your ArgoCD cluster:
     kubectl apply -f app-of-apps.yaml`,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "recipe",
				Aliases:  []string{"r"},
				Required: true,
				Usage: `Path/URI to previously generated recipe from which to build the bundle.
	Supports: file paths, HTTP/HTTPS URLs, or ConfigMap URIs (cm://namespace/name).`,
			},
			&cli.StringFlag{
				Name:    "output",
				Aliases: []string{"o"},
				Value:   ".",
				Usage:   "Output directory path for the generated Helm umbrella chart",
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
				Name:    "deployer",
				Aliases: []string{"d"},
				Value:   "",
				Usage:   "Deployment method: '' (default Helm umbrella chart) or 'argocd' (App of Apps pattern)",
			},
			&cli.StringFlag{
				Name:  "repo",
				Value: "",
				Usage: "Git repository URL for ArgoCD applications (only used with --deployer argocd)",
			},
			// Output format flag
			&cli.StringFlag{
				Name:    "output-format",
				Aliases: []string{"F"},
				Value:   outputFormatDir,
				Usage:   "Output format: 'dir' (local directory) or 'oci' (OCI Image Layout)",
			},
			// OCI registry flags (used when output-format is oci)
			&cli.StringFlag{
				Name:  "registry",
				Usage: "OCI registry host for image reference (e.g., ghcr.io, localhost:5000)",
			},
			&cli.StringFlag{
				Name:  "repository",
				Usage: "OCI repository path for image reference (e.g., nvidia/cns-bundle)",
			},
			&cli.StringFlag{
				Name:  "tag",
				Usage: fmt.Sprintf("OCI image tag (default: %s)", defaultOCITag),
			},
			// Push flag - controls whether to push to remote registry
			&cli.BoolFlag{
				Name:  "push",
				Usage: "Push OCI artifact to remote registry (requires --output-format=oci)",
			},
			&cli.BoolFlag{
				Name:  "insecure-tls",
				Usage: "Skip TLS certificate verification for OCI registry",
			},
			&cli.BoolFlag{
				Name:  "plain-http",
				Usage: "Use HTTP instead of HTTPS for OCI registry (for local development)",
			},
			kubeconfigFlag,
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			opts, err := parseBundleCmdOptions(cmd)
			if err != nil {
				return err
			}

			outputType := "umbrella chart"
			if opts.deployer == deployerArgoCD {
				outputType = "ArgoCD applications"
			}
			slog.Info("generating bundle",
				slog.String("type", outputType),
				slog.String("recipe", opts.recipeFilePath),
				slog.String("output", opts.outputDir),
				slog.String("output-format", opts.outputFormat),
			)

			// Load recipe from file/URL/ConfigMap
			rec, err := serializer.FromFileWithKubeconfig[recipe.RecipeResult](opts.recipeFilePath, opts.kubeconfig)
			if err != nil {
				slog.Error("failed to load recipe file", "error", err, "path", opts.recipeFilePath)
				return err
			}

			// Create bundler with config
			cfg := config.NewConfig(
				config.WithVersion(version),
				config.WithDeployer(opts.deployer),
				config.WithRepoURL(opts.repoURL),
				config.WithValueOverrides(opts.valueOverrides),
				config.WithSystemNodeSelector(opts.systemNodeSelector),
				config.WithSystemNodeTolerations(opts.systemNodeTolerations),
				config.WithAcceleratedNodeSelector(opts.acceleratedNodeSelector),
				config.WithAcceleratedNodeTolerations(opts.acceleratedNodeTolerations),
			)

			b, err := bundler.NewWithConfig(cfg)
			if err != nil {
				slog.Error("failed to create bundler", "error", err)
				return err
			}

			// Generate bundle
			out, err := b.Make(ctx, rec, opts.outputDir)
			if err != nil {
				slog.Error("bundle generation failed", "error", err)
				return err
			}

			slog.Info("bundle generated",
				"type", outputType,
				"files", out.TotalFiles,
				"size_bytes", out.TotalSize,
				"duration_sec", out.TotalDuration.Seconds(),
				"output_dir", out.OutputDir,
			)

			// Print deployment instructions (only for dir output)
			if opts.outputFormat == outputFormatDir {
				printBundleDeploymentInstructions(opts.deployer, opts.repoURL, out)
			}

			// Package as OCI artifact when output-format is oci
			if opts.outputFormat == outputFormatOCI {
				if ociErr := handleOCIOutput(ctx, ociConfig{
					sourceDir:   opts.outputDir,
					outputDir:   opts.outputDir,
					registry:    opts.registryHost,
					repository:  opts.repository,
					tag:         opts.tag,
					push:        opts.push,
					plainHTTP:   opts.plainHTTP,
					insecureTLS: opts.insecureTLS,
				}, out.Results); ociErr != nil {
					return ociErr
				}
			}

			return nil
		},
	}
}

// printBundleDeploymentInstructions prints user-friendly deployment instructions.
func printBundleDeploymentInstructions(deployer, repoURL string, out *result.Output) {
	if deployer == deployerArgoCD {
		fmt.Printf("\nArgoCD applications generated successfully!\n")
		fmt.Printf("Output directory: %s\n", out.OutputDir)
		fmt.Printf("Files generated: %d\n", out.TotalFiles)
		fmt.Printf("\nTo deploy:\n")
		fmt.Printf("  1. Push the generated files to your GitOps repository\n")
		if repoURL == "" {
			fmt.Printf("  2. Update app-of-apps.yaml with your repository URL\n")
			fmt.Printf("  3. Apply to your cluster:\n")
		} else {
			fmt.Printf("  2. Apply to your cluster:\n")
		}
		fmt.Printf("     kubectl apply -f %s/app-of-apps.yaml\n", out.OutputDir)
	} else {
		fmt.Printf("\nUmbrella chart generated successfully!\n")
		fmt.Printf("Output directory: %s\n", out.OutputDir)
		fmt.Printf("Files generated: %d\n", out.TotalFiles)
		fmt.Printf("\nTo deploy:\n")
		fmt.Printf("  cd %s\n", out.OutputDir)
		fmt.Printf("  helm dependency update\n")
		fmt.Printf("  helm install cns-stack . -n cns-system --create-namespace\n")
	}
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

	// Default tag if not provided
	imageTag := cfg.tag
	if imageTag == "" {
		imageTag = defaultOCITag
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

	// Update results with OCI metadata (pushed=false initially, updated after successful push)
	for i := range results {
		if results[i].Success {
			results[i].SetOCIMetadata(packageResult.Digest, packageResult.Reference, false)
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

		// Mark results as pushed after successful push
		for i := range results {
			if results[i].Success {
				results[i].Pushed = true
			}
		}
	}

	return nil
}
