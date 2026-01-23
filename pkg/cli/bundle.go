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

	"github.com/distribution/reference"
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

// OCI output constants.
const (
	defaultOCITag = "latest"
	// OCIURIScheme is the URI scheme for OCI registry output.
	OCIURIScheme = "oci://"
)

// bundleCmdOptions holds parsed options for the bundle command.
type bundleCmdOptions struct {
	recipeFilePath             string
	outputDir                  string
	kubeconfig                 string
	deployer                   config.DeployerType
	repoURL                    string
	valueOverrides             map[string]map[string]string
	systemNodeSelector         map[string]string
	systemNodeTolerations      []corev1.Toleration
	acceleratedNodeSelector    map[string]string
	acceleratedNodeTolerations []corev1.Toleration
	// Output target fields (parsed from --output flag)
	outputIsOCI   bool   // true if oci:// scheme detected
	ociRegistry   string // parsed from oci:// URI
	ociRepository string // parsed from oci:// URI
	ociTag        string // parsed from oci:// URI (default: "latest")
	plainHTTP     bool
	insecureTLS   bool
}

// parseOutputTarget parses the --output flag value to detect OCI URI or local directory.
// For OCI URIs (oci://registry/repository:tag), it extracts the components.
// For plain paths, it treats them as local directories.
func parseOutputTarget(target string) (isOCI bool, registry, repository, tag, dirPath string, err error) {
	if !strings.HasPrefix(target, OCIURIScheme) {
		return false, "", "", "", target, nil
	}

	// Strip oci:// and parse as standard image reference
	ref, err := reference.ParseNormalizedNamed(strings.TrimPrefix(target, OCIURIScheme))
	if err != nil {
		return false, "", "", "", "", fmt.Errorf("invalid OCI reference: %w", err)
	}

	// Extract components using the reference package
	registry = reference.Domain(ref)
	repository = reference.Path(ref)

	if tagged, ok := ref.(reference.Tagged); ok {
		tag = tagged.Tag()
	} else {
		tag = defaultOCITag
	}

	return true, registry, repository, tag, "", nil
}

// parseBundleCmdOptions parses and validates command options.
func parseBundleCmdOptions(cmd *cli.Command) (*bundleCmdOptions, error) {
	opts := &bundleCmdOptions{
		recipeFilePath: cmd.String("recipe"),
		kubeconfig:     cmd.String("kubeconfig"),
		repoURL:        cmd.String("repo"),
		insecureTLS:    cmd.Bool("insecure-tls"),
		plainHTTP:      cmd.Bool("plain-http"),
	}

	// Parse and validate deployer flag using strongly-typed parser
	deployerStr := cmd.String("deployer")
	if deployerStr == "" {
		opts.deployer = config.DeployerHelm
	} else {
		deployer, err := config.ParseDeployerType(deployerStr)
		if err != nil {
			return nil, fmt.Errorf("invalid --deployer value: %w", err)
		}
		opts.deployer = deployer
	}

	// Parse output target (detects oci:// URI or local directory)
	outputTarget := cmd.String("output")
	isOCI, registry, repository, tag, dirPath, err := parseOutputTarget(outputTarget)
	if err != nil {
		return nil, fmt.Errorf("invalid --output value: %w", err)
	}

	opts.outputIsOCI = isOCI
	if isOCI {
		opts.ociRegistry = registry
		opts.ociRepository = repository
		opts.ociTag = tag
		// For OCI output, we still need a temp directory for bundle generation
		opts.outputDir = "."
		// Validate registry and repository format
		if valErr := oci.ValidateRegistryReference(registry, repository); valErr != nil {
			return nil, fmt.Errorf("invalid --output OCI URI: %w", valErr)
		}
	} else {
		opts.outputDir = dirPath
	}

	// Parse value overrides from --set flags
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
  - checksums.txt: SHA256 checksums of generated files

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

Package and push bundle to OCI registry:
  cnsctl bundle --recipe recipe.yaml --output oci://ghcr.io/nvidia/cns-bundle:v1.0.0
`,
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
				Usage: `Output target: local directory path or OCI registry URI.
	For local output: ./my-bundle or /tmp/bundle
	For OCI registry: oci://ghcr.io/nvidia/bundle:v1.0.0`,
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
				Value:   string(config.DeployerHelm),
				Usage:   fmt.Sprintf("Deployment method: %v", config.GetDeployerTypes()),
			},
			&cli.StringFlag{
				Name:  "repo",
				Value: "",
				Usage: "Git repository URL for ArgoCD applications (only used with --deployer argocd)",
			},
			kubeconfigFlag,
			// OCI registry connection flags (used when --output is oci://...)
			&cli.BoolFlag{
				Name:  "insecure-tls",
				Usage: "Skip TLS certificate verification for OCI registry",
			},
			&cli.BoolFlag{
				Name:  "plain-http",
				Usage: "Use HTTP instead of HTTPS for OCI registry (for local development)",
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			opts, err := parseBundleCmdOptions(cmd)
			if err != nil {
				return err
			}

			outputType := "Helm umbrella chart"
			if opts.deployer == config.DeployerArgoCD {
				outputType = "ArgoCD applications"
			}
			slog.Info("generating bundle",
				slog.String("deployer", opts.deployer.String()),
				slog.String("type", outputType),
				slog.String("recipe", opts.recipeFilePath),
				slog.String("output", opts.outputDir),
				slog.Bool("oci", opts.outputIsOCI),
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
			if !opts.outputIsOCI {
				printBundleDeploymentInstructions(opts.deployer, opts.repoURL, out)
			}

			// Package and push as OCI artifact when output is oci://
			if opts.outputIsOCI {
				if ociErr := handleOCIOutput(ctx, ociConfig{
					sourceDir:   opts.outputDir,
					outputDir:   opts.outputDir,
					registry:    opts.ociRegistry,
					repository:  opts.ociRepository,
					tag:         opts.ociTag,
					version:     version,
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
func printBundleDeploymentInstructions(deployer config.DeployerType, repoURL string, out *result.Output) {
	if deployer == config.DeployerArgoCD {
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
		fmt.Printf("  helm install cns-stack .\n")
	}
}

// ociConfig holds configuration for OCI operations.
type ociConfig struct {
	sourceDir   string
	outputDir   string
	registry    string
	repository  string
	tag         string
	version     string
	plainHTTP   bool
	insecureTLS bool
}

// handleOCIOutput packages the bundle as an OCI artifact and pushes it to the registry.
// OCI metadata (digest, reference) is populated on results when --output=oci://... is used.
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

	slog.Info("packaging and pushing bundle as OCI artifact",
		"registry", cfg.registry,
		"repository", cfg.repository,
		"tag", imageTag,
	)

	// Package locally first
	packageResult, err := oci.Package(ctx, oci.PackageOptions{
		SourceDir:  absSourceDir,
		OutputDir:  absOutputDir,
		Registry:   cfg.registry,
		Repository: cfg.repository,
		Tag:        imageTag,
		Annotations: map[string]string{
			"org.opencontainers.image.version": cfg.version,
			"org.opencontainers.image.vendor":  "NVIDIA",
			"org.opencontainers.image.title":   "CNS Bundle",
			"org.opencontainers.image.source":  "https://github.com/NVIDIA/cloud-native-stack",
		},
	})
	if err != nil {
		return fmt.Errorf("failed to package OCI artifact: %w", err)
	}

	slog.Info("OCI artifact packaged locally",
		"reference", packageResult.Reference,
		"digest", packageResult.Digest,
		"store_path", packageResult.StorePath,
	)

	// Push to remote registry
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

	// Update results with OCI metadata
	for i := range results {
		if results[i].Success {
			results[i].SetOCIMetadata(pushResult.Digest, pushResult.Reference, true)
		}
	}

	return nil
}
