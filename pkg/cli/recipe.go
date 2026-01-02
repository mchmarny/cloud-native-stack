/*
Copyright © 2025 NVIDIA Corporation
SPDX-License-Identifier: Apache-2.0
*/
package cli

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/urfave/cli/v3"

	"github.com/NVIDIA/cloud-native-stack/pkg/recipe"
	ver "github.com/NVIDIA/cloud-native-stack/pkg/recipe/version"
	"github.com/NVIDIA/cloud-native-stack/pkg/serializer"
	"github.com/NVIDIA/cloud-native-stack/pkg/snapshotter"
)

func recipeCmd() *cli.Command {
	return &cli.Command{
		Name:                  "recipe",
		EnableShellCompletion: true,
		Usage:                 "Generate configuration recipe for a given environment",
		Description: `Generate configuration recipe based on specified environment parameters including:
  - Operating system and version
  - Kernel version
  - Managed service context
  - Kubernetes cluster version
  - GPU type
  - Workload intent

The recipe can be output in JSON, YAML, or table format.`,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name: "os",
				Usage: fmt.Sprintf("Operating system family (supported values: %s)",
					recipe.SupportedOSFamilies()),
			},
			&cli.StringFlag{
				Name:  "osv",
				Usage: "Operating system version (e.g., 22.04)",
			},
			&cli.StringFlag{
				Name:  "kernel",
				Usage: "Running kernel version (e.g., 5.15.0)",
			},
			&cli.StringFlag{
				Name: "service",
				Usage: fmt.Sprintf("Managed Kubernetes service type (supported values: %s)",
					recipe.SupportedServiceTypes()),
			},
			&cli.StringFlag{
				Name:  "k8s",
				Usage: "Kubernetes cluster version (e.g., v1.25.4)",
			},
			&cli.StringFlag{
				Name: "gpu",
				Usage: fmt.Sprintf("GPU type (supported values: %s)",
					recipe.SupportedGPUTypes()),
			},
			&cli.StringFlag{
				Name:  "intent",
				Value: recipe.IntentTraining.String(),
				Usage: fmt.Sprintf("Workload intent for a given configuration (supported values: %s)",
					recipe.SupportedIntentTypes()),
			},
			&cli.BoolFlag{
				Name:  "context",
				Usage: "Includes configuration metadata in the response",
			},
			&cli.StringFlag{
				Name:    "snapshot",
				Aliases: []string{"f"},
				Usage: `Path/URI to previously generated configuration snapshot from which to build the recipe.
	Supports: file paths, HTTP/HTTPS URLs, or ConfigMap URIs (cm://namespace/name).
	If provided, all other query flags with the exception of intent are ignored.`,
			},
			outputFlag,
			formatFlag,
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			// Parse output format
			outFormat := serializer.Format(cmd.String("format"))
			if outFormat.IsUnknown() {
				return fmt.Errorf("unknown output format: %q", outFormat)
			}

			// Parse intent
			intentStr := cmd.String("intent")
			intent := recipe.IntentType(intentStr)
			if !intent.IsValid() {
				return fmt.Errorf("invalid intent type: %q", intentStr)
			}

			var rec *recipe.Recipe

			// Create builder
			builder := recipe.NewBuilder(
				recipe.WithVersion(version),
			)

			// Load snapshot
			snapFilePath := cmd.String("snapshot")
			if snapFilePath != "" {
				snap, err := serializer.FromFile[snapshotter.Snapshot](snapFilePath)
				if err != nil {
					return fmt.Errorf("failed to load snapshot from %q: %w", snapFilePath, err)
				}

				rec, err = builder.BuildFromSnapshot(ctx, intent, snap)
				if err != nil {
					return fmt.Errorf("error building recipe from snapshot: %w", err)
				}
			} else {
				q, err := buildQueryFromCmd(cmd)
				if err != nil {
					return fmt.Errorf("error parsing recipe input parameter: %w", err)
				}

				rec, err = builder.BuildFromQuery(ctx, q)
				if err != nil {
					return fmt.Errorf("error building recipe from query: %w", err)
				}
			}

			ser := serializer.NewFileWriterOrStdout(outFormat, cmd.String("output"))
			defer func() {
				if closer, ok := ser.(interface{ Close() error }); ok {
					if err := closer.Close(); err != nil {
						slog.Warn("failed to close serializer", "error", err)
					}
				}
			}()

			return ser.Serialize(rec)
		},
	}
}

// buildQueryFromCmd constructs a recipe.Query from CLI command.
func buildQueryFromCmd(cmd *cli.Command) (*recipe.Query, error) {
	q := &recipe.Query{}

	if recOs := cmd.String("os"); recOs != "" {
		q.Os = recipe.OsFamily(recOs)
		if !q.Os.IsValid() {
			return nil, fmt.Errorf("os: %q, supported values: %v", recOs, recipe.SupportedOSFamilies())
		}
	}
	if recOsVersion := cmd.String("osv"); recOsVersion != "" {
		v, err := ver.ParseVersion(recOsVersion)
		if err != nil {
			if errors.Is(err, ver.ErrNegativeComponent) {
				return nil, fmt.Errorf("os version cannot contain negative numbers: %s", recOsVersion)
			}
			return nil, fmt.Errorf("invalid os version %q: %w", recOsVersion, err)
		}
		q.OsVersion = &v
	}
	if recKernel := cmd.String("kernel"); recKernel != "" {
		v, err := ver.ParseVersion(recKernel)
		if err != nil {
			if errors.Is(err, ver.ErrNegativeComponent) {
				return nil, fmt.Errorf("kernel version cannot contain negative numbers: %s", recKernel)
			}
			return nil, fmt.Errorf("invalid kernel version %q: %w", recKernel, err)
		}
		q.Kernel = &v
	}
	if recService := cmd.String("service"); recService != "" {
		q.Service = recipe.ServiceType(recService)
		if !q.Service.IsValid() {
			return nil, fmt.Errorf("service: %q, supported values: %v", recService, recipe.SupportedServiceTypes())
		}
	}

	if recK8s := cmd.String("k8s"); recK8s != "" {
		v, err := ver.ParseVersion(recK8s)
		if err != nil {
			if errors.Is(err, ver.ErrNegativeComponent) {
				return nil, fmt.Errorf("kubernetes version cannot contain negative numbers: %s", recK8s)
			}
			return nil, fmt.Errorf("invalid kubernetes version %q: %w", recK8s, err)
		}
		q.K8s = &v
	}
	if recGPU := cmd.String("gpu"); recGPU != "" {
		q.GPU = recipe.GPUType(recGPU)
		if !q.GPU.IsValid() {
			return nil, fmt.Errorf("gpu: %q, supported values: %v", recGPU, recipe.SupportedGPUTypes())
		}
	}
	if recIntent := cmd.String("intent"); recIntent != "" {
		q.Intent = recipe.IntentType(recIntent)
		if !q.Intent.IsValid() {
			return nil, fmt.Errorf("intent: %q, supported values: %v", recIntent, recipe.SupportedIntentTypes())
		}
	}

	q.IncludeContext = cmd.Bool("context")

	return q, nil
}
