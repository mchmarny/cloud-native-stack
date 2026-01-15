/*
Copyright © 2025 NVIDIA Corporation
SPDX-License-Identifier: Apache-2.0
*/
package cli

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/urfave/cli/v3"

	"github.com/NVIDIA/cloud-native-stack/pkg/recipe"
	"github.com/NVIDIA/cloud-native-stack/pkg/serializer"
	"github.com/NVIDIA/cloud-native-stack/pkg/snapshotter"
	"github.com/NVIDIA/cloud-native-stack/pkg/validator"
)

func validateCmd() *cli.Command {
	return &cli.Command{
		Name:                  "validate",
		EnableShellCompletion: true,
		Usage:                 "Validate cluster configuration against recipe constraints",
		Description: `Validate a system snapshot against the constraints defined in a recipe.

This command compares actual system measurements from a snapshot against the
expected constraints defined in a recipe file. It reports which constraints
pass, fail, or cannot be evaluated.

# Constraint Format

Constraints in recipes use fully qualified measurement paths:
  {Type}.{Subtype}.{Key}

Examples:
  K8s.server.version           - Kubernetes server version
  OS.release.ID                - Operating system identifier
  OS.release.VERSION_ID        - OS version
  OS.sysctl./proc/sys/kernel/osrelease - Kernel version

# Supported Operators

Constraint values can use comparison operators:
  ">= 1.32.4"  - Greater than or equal (version comparison)
  "<= 1.33"    - Less than or equal (version comparison)
  "> 1.30"     - Greater than (version comparison)
  "< 2.0"      - Less than (version comparison)
  "== ubuntu"  - Exact match
  "!= rhel"    - Not equal
  "ubuntu"     - Exact string match (no operator)

# Examples

Validate a snapshot against a recipe:
  cnsctl validate --recipe recipe.yaml --snapshot snapshot.yaml

Load snapshot from ConfigMap (results to stdout):
  cnsctl validate --recipe recipe.yaml --snapshot cm://gpu-operator/cns-snapshot

Output validation result to a file:
  cnsctl validate -r recipe.yaml -s snapshot.yaml -o result.yaml

Fail the command if any constraint fails (useful for CI/CD):
  cnsctl validate -r recipe.yaml -s snapshot.yaml --fail-on-error`,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "recipe",
				Aliases:  []string{"r"},
				Required: true,
				Usage: `Path/URI to recipe file containing constraints to validate.
	Supports: file paths, HTTP/HTTPS URLs, or ConfigMap URIs (cm://namespace/name).`,
			},
			&cli.StringFlag{
				Name:     "snapshot",
				Aliases:  []string{"s"},
				Required: true,
				Usage: `Path/URI to snapshot file containing actual system measurements.
	Supports: file paths, HTTP/HTTPS URLs, or ConfigMap URIs (cm://namespace/name).`,
			},
			&cli.BoolFlag{
				Name:  "fail-on-error",
				Usage: "Exit with non-zero status if any constraint fails validation",
			},
			outputFlag,
			formatFlag,
			kubeconfigFlag,
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			// Parse output format
			outFormat := serializer.Format(cmd.String("format"))
			if outFormat.IsUnknown() {
				return fmt.Errorf("unknown output format: %q", outFormat)
			}

			recipeFilePath := cmd.String("recipe")
			snapshotFilePath := cmd.String("snapshot")
			kubeconfig := cmd.String("kubeconfig")
			failOnError := cmd.Bool("fail-on-error")

			slog.Info("loading recipe", "uri", recipeFilePath)

			// Load recipe
			rec, err := serializer.FromFileWithKubeconfig[recipe.RecipeResult](recipeFilePath, kubeconfig)
			if err != nil {
				return fmt.Errorf("failed to load recipe from %q: %w", recipeFilePath, err)
			}

			slog.Info("loading snapshot", "uri", snapshotFilePath)

			// Load snapshot
			snap, err := serializer.FromFileWithKubeconfig[snapshotter.Snapshot](snapshotFilePath, kubeconfig)
			if err != nil {
				return fmt.Errorf("failed to load snapshot from %q: %w", snapshotFilePath, err)
			}

			slog.Info("validating constraints",
				"recipe", recipeFilePath,
				"snapshot", snapshotFilePath,
				"constraints", len(rec.Constraints))

			// Create validator
			v := validator.New(
				validator.WithVersion(version),
			)

			// Validate
			result, err := v.Validate(ctx, rec, snap)
			if err != nil {
				return fmt.Errorf("validation failed: %w", err)
			}

			// Set source information
			result.RecipeSource = recipeFilePath
			result.SnapshotSource = snapshotFilePath

			// Serialize output
			output := cmd.String("output")
			ser := serializer.NewFileWriterOrStdout(outFormat, output)
			defer func() {
				if closer, ok := ser.(interface{ Close() error }); ok {
					if err := closer.Close(); err != nil {
						slog.Warn("failed to close serializer", "error", err)
					}
				}
			}()

			if err := ser.Serialize(ctx, result); err != nil {
				return fmt.Errorf("failed to serialize validation result: %w", err)
			}

			slog.Info("validation completed",
				"status", result.Summary.Status,
				"passed", result.Summary.Passed,
				"failed", result.Summary.Failed,
				"skipped", result.Summary.Skipped,
				"duration", result.Summary.Duration)

			// Check if we should fail on validation errors
			if failOnError && result.Summary.Status == validator.ValidationStatusFail {
				return fmt.Errorf("validation failed: %d constraint(s) did not pass", result.Summary.Failed)
			}

			return nil
		},
	}
}
