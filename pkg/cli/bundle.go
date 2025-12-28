/*
Copyright © 2025 NVIDIA Corporation
SPDX-License-Identifier: Apache-2.0
*/
package cli

import (
	"context"
	"log/slog"

	"github.com/NVIDIA/cloud-native-stack/pkg/bundler"
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
				Name:    "recipe",
				Aliases: []string{"f"},
				Usage:   "File path to previously generated recipe file from which to build the bundle.",
			},
			&cli.StringFlag{
				Name:     "output",
				Aliases:  []string{"o"},
				Value:    ".",
				Required: true,
				Usage:    "output directory path",
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			recipeFilePath := cmd.String("recipe")
			outputDir := cmd.String("output")

			slog.Info("generating bundle",
				"recipeFilePath", recipeFilePath,
				"outputDir", outputDir,
			)

			rec, err := serializer.FromFile[recipe.Recipe](recipeFilePath)
			if err != nil {
				slog.Error("failed to load recipe file", "error", err, "path", recipeFilePath)
				return err
			}

			// Create bundler instance using default configuration
			b := bundler.New()

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

			return nil
		},
	}
}
