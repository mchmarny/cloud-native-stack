/*
Copyright © 2025 NVIDIA Corporation
SPDX-License-Identifier: Apache-2.0
*/
package cli

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/NVIDIA/cloud-native-stack/pkg/bundler"
	"github.com/NVIDIA/cloud-native-stack/pkg/bundler/common"
	"github.com/NVIDIA/cloud-native-stack/pkg/recipe"
	"github.com/NVIDIA/cloud-native-stack/pkg/serializer"
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
				Aliases:  []string{"f"},
				Required: true,
				Usage:    "File path or URL to previously generated recipe file from which to build the common.",
			},
			&cli.StringSliceFlag{
				Name:    "bundlers",
				Aliases: []string{"b"},
				Usage: fmt.Sprintf(`Types of bundlers to execute (supported types: %s). 
	If not specified, all supported bundlers are executed.`, common.SupportedBundleTypesAsStrings()),
			},
			&cli.StringFlag{
				Name:    "output",
				Aliases: []string{"o"},
				Value:   ".",
				Usage:   "output directory path",
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			recipeFilePath := cmd.String("recipe")
			outputDir := cmd.String("output")
			bundlerTypesStr := cmd.StringSlice("bundlers")

			// Parse bundler types
			var bundlerTypes []common.BundleType
			for _, t := range bundlerTypesStr {
				bt, err := common.ParseType(t)
				if err != nil {
					return fmt.Errorf("invalid bundler type '%s': %w", t, err)
				}
				bundlerTypes = append(bundlerTypes, bt)
			}

			slog.Info("generating bundle",
				slog.String("recipeFilePath", recipeFilePath),
				slog.String("outputDir", outputDir),
				slog.Any("bundlerTypes", bundlerTypes),
			)

			rec, err := serializer.FromFile[recipe.Recipe](recipeFilePath)
			if err != nil {
				slog.Error("failed to load recipe file", "error", err, "path", recipeFilePath)
				return err
			}

			// Create bundler instance
			b := bundler.New(
				// If bundler types are not specified, all supported bundlers are used.
				// An empty or nil slice means all bundlers as well.
				bundler.WithBundlerTypes(bundlerTypes),
			)

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
