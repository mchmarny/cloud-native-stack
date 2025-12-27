/*
Copyright © 2025 NVIDIA Corporation
SPDX-License-Identifier: Apache-2.0
*/
package cli

import (
	"context"
	"errors"
	"fmt"

	"github.com/urfave/cli/v3"

	"github.com/NVIDIA/cloud-native-stack/pkg/recipe"
	ver "github.com/NVIDIA/cloud-native-stack/pkg/recipe/version"
	"github.com/NVIDIA/cloud-native-stack/pkg/serializer"
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
				Name:  "os",
				Usage: "Operating system family (e.g., ubuntu, cos)",
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
				Name:  "service",
				Usage: "Managed Kubernetes service type (e.g., eks, gke, or self-managed)",
			},
			&cli.StringFlag{
				Name:  "k8s",
				Usage: "Kubernetes cluster version (e.g., v1.25.4)",
			},
			&cli.StringFlag{
				Name:  "gpu",
				Usage: "GPU type (e.g., H100, GB200)",
			},
			&cli.StringFlag{
				Name:  "intent",
				Usage: "Workload intent for a given configuration (e.g., training or inference)",
			},
			&cli.BoolFlag{
				Name:  "context",
				Usage: "Include metadata in the response for given configuration",
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

			q, err := buildQueryFromCmd(cmd)
			if err != nil {
				return fmt.Errorf("error parsing recipe input parameter: %w", err)
			}

			rec, err := recipe.BuildRecipe(ctx, q)
			if err != nil {
				return fmt.Errorf("error building recipe: %w", err)
			}

			return serializer.NewFileWriterOrStdout(outFormat, cmd.String("output")).Serialize(rec)
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
