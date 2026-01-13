/*
Copyright © 2025 NVIDIA Corporation
SPDX-License-Identifier: Apache-2.0
*/
package cli

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"strings"

	"github.com/urfave/cli/v3"

	"github.com/NVIDIA/cloud-native-stack/pkg/measurement"
	"github.com/NVIDIA/cloud-native-stack/pkg/recipe"
	"github.com/NVIDIA/cloud-native-stack/pkg/serializer"
	"github.com/NVIDIA/cloud-native-stack/pkg/snapshotter"
)

// DetectionSource describes where a criteria value was detected from.
type DetectionSource struct {
	Value      string // The detected value
	Source     string // Human-readable source description
	RawValue   string // The raw value from the snapshot (e.g., full version string)
	Overridden bool   // Whether this was overridden by CLI flag
}

// CriteriaDetection holds criteria with detection sources for transparency.
type CriteriaDetection struct {
	Service     *DetectionSource
	Accelerator *DetectionSource
	OS          *DetectionSource
	Intent      *DetectionSource
	Nodes       *DetectionSource
}

// PrintDetection outputs the detected criteria to the given writer.
func (cd *CriteriaDetection) PrintDetection(w io.Writer) {
	fmt.Fprintln(w, "Detected criteria from snapshot:")
	printDetectionField(w, "service", cd.Service)
	printDetectionField(w, "accelerator", cd.Accelerator)
	printDetectionField(w, "os", cd.OS)
	printDetectionField(w, "intent", cd.Intent)
	printDetectionField(w, "nodes", cd.Nodes)
	fmt.Fprintln(w)
}

func printDetectionField(w io.Writer, name string, ds *DetectionSource) {
	if ds == nil {
		fmt.Fprintf(w, "  %-12s (not detected)\n", name+":")
		return
	}

	switch {
	case ds.Overridden:
		fmt.Fprintf(w, "  %-12s %-12s (overridden by --%s flag)\n", name+":", ds.Value, name)
	case ds.RawValue != "" && ds.RawValue != ds.Value:
		fmt.Fprintf(w, "  %-12s %-12s (from %s: %s)\n", name+":", ds.Value, ds.Source, ds.RawValue)
	default:
		fmt.Fprintf(w, "  %-12s %-12s (from %s)\n", name+":", ds.Value, ds.Source)
	}
}

func recipeCmd() *cli.Command {
	return &cli.Command{
		Name:                  "recipe",
		EnableShellCompletion: true,
		Usage:                 "Generate configuration recipe for a given set of environment parameters.",
		Description: `Generate configuration recipe based on specified environment parameters including:
  - Kubernetes service type (eks, gke, aks, oke, self-managed)
  - Accelerator type (h100, gb200, a100, l40)
  - Workload intent (training, inference)
  - GPU node operating system (ubuntu, rhel, cos, amazonlinux)
  - Number of GPU nodes in the cluster

The recipe returns a list of components with deployment order based on dependencies.
Output can be in JSON or YAML format.`,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "service",
				Usage: fmt.Sprintf("Kubernetes service type (e.g. %s)", strings.Join(recipe.GetCriteriaServiceTypes(), ", ")),
			},
			&cli.StringFlag{
				Name:    "accelerator",
				Aliases: []string{"gpu"},
				Usage:   fmt.Sprintf("Accelerator/GPU type (e.g. %s)", strings.Join(recipe.GetCriteriaAcceleratorTypes(), ", ")),
			},
			&cli.StringFlag{
				Name:  "intent",
				Usage: fmt.Sprintf("Workload intent (e.g. %s)", strings.Join(recipe.GetCriteriaIntentTypes(), ", ")),
			},
			&cli.StringFlag{
				Name:  "os",
				Usage: fmt.Sprintf("Operating system type of the GPU node (e.g. %s)", strings.Join(recipe.GetCriteriaOSTypes(), ", ")),
			},
			&cli.IntFlag{
				Name:  "nodes",
				Usage: "Number of worker/GPU nodes in the cluster",
			},
			&cli.StringFlag{
				Name:    "snapshot",
				Aliases: []string{"f"},
				Usage: `Path/URI to previously generated configuration snapshot.
	Supports: file paths, HTTP/HTTPS URLs, or ConfigMap URIs (cm://namespace/name).
	If provided, criteria are extracted from the snapshot.`,
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

			// Create builder
			builder := recipe.NewBuilder(
				recipe.WithVersion(version),
			)

			var result *recipe.RecipeResult
			var err error

			// Check if using snapshot
			snapFilePath := cmd.String("snapshot")
			if snapFilePath != "" {
				slog.Info("loading snapshot from", "uri", snapFilePath)
				snap, loadErr := serializer.FromFileWithKubeconfig[snapshotter.Snapshot](snapFilePath, cmd.String("kubeconfig"))
				if loadErr != nil {
					return fmt.Errorf("failed to load snapshot from %q: %w", snapFilePath, loadErr)
				}

				// Extract criteria from snapshot with detection sources
				criteria, detection := extractCriteriaFromSnapshot(snap)

				// Apply CLI overrides and track them
				if applyErr := applyCriteriaOverrides(cmd, criteria, detection); applyErr != nil {
					return applyErr
				}

				// Print detected criteria for transparency (to stderr so it doesn't interfere with output)
				detection.PrintDetection(cmd.ErrWriter)

				slog.Info("building recipe from snapshot", "criteria", criteria.String())
				result, err = builder.BuildFromCriteria(ctx, criteria)
			} else {
				// Build criteria from CLI flags
				criteria, buildErr := buildCriteriaFromCmd(cmd)
				if buildErr != nil {
					return fmt.Errorf("error parsing criteria: %w", buildErr)
				}

				slog.Info("building recipe from criteria", "criteria", criteria.String())
				result, err = builder.BuildFromCriteria(ctx, criteria)
			}

			if err != nil {
				return fmt.Errorf("error building recipe: %w", err)
			}

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
				return fmt.Errorf("failed to serialize recipe: %w", err)
			}

			slog.Info("recipe generation completed",
				"output", output,
				"components", len(result.ComponentRefs),
				"overlays", len(result.Metadata.AppliedOverlays))

			return nil
		},
	}
}

// buildCriteriaFromCmd constructs a recipe.Criteria from CLI command flags.
func buildCriteriaFromCmd(cmd *cli.Command) (*recipe.Criteria, error) {
	var opts []recipe.CriteriaOption

	if s := cmd.String("service"); s != "" {
		opts = append(opts, recipe.WithCriteriaService(s))
	}
	if s := cmd.String("accelerator"); s != "" {
		opts = append(opts, recipe.WithCriteriaAccelerator(s))
	}
	if s := cmd.String("intent"); s != "" {
		opts = append(opts, recipe.WithCriteriaIntent(s))
	}
	if s := cmd.String("os"); s != "" {
		opts = append(opts, recipe.WithCriteriaOS(s))
	}
	if n := cmd.Int("nodes"); n > 0 {
		opts = append(opts, recipe.WithCriteriaNodes(n))
	}

	return recipe.BuildCriteria(opts...)
}

// detectAcceleratorFromModel detects accelerator type from a model string.
func detectAcceleratorFromModel(modelStr string) recipe.CriteriaAcceleratorType {
	switch {
	case containsIgnoreCase(modelStr, "gb200"):
		return recipe.CriteriaAcceleratorGB200
	case containsIgnoreCase(modelStr, "h100"):
		return recipe.CriteriaAcceleratorH100
	case containsIgnoreCase(modelStr, "a100"):
		return recipe.CriteriaAcceleratorA100
	case containsIgnoreCase(modelStr, "l40"):
		return recipe.CriteriaAcceleratorL40
	default:
		return ""
	}
}

// extractCriteriaFromSnapshot extracts criteria from a snapshot.
// This maps snapshot measurements to criteria fields and returns detection sources.
func extractCriteriaFromSnapshot(snap *snapshotter.Snapshot) (*recipe.Criteria, *CriteriaDetection) {
	criteria := recipe.NewCriteria()
	detection := &CriteriaDetection{}

	if snap == nil {
		return criteria, detection
	}

	for _, m := range snap.Measurements {
		if m == nil {
			continue
		}

		switch m.Type {
		case measurement.TypeK8s:
			extractK8sCriteria(m, criteria, detection)
		case measurement.TypeGPU:
			extractGPUCriteria(m, criteria, detection)
		case measurement.TypeOS:
			extractOSCriteria(m, criteria, detection)
		case measurement.TypeSystemD:
			continue
		}
	}

	return criteria, detection
}

// extractK8sCriteria extracts Kubernetes service type from measurements.
func extractK8sCriteria(m *measurement.Measurement, criteria *recipe.Criteria, detection *CriteriaDetection) {
	for _, st := range m.Subtypes {
		if st.Name != "server" {
			continue
		}
		// Try direct "service" field first
		if svcType, ok := st.Data["service"]; ok {
			if parsed, err := recipe.ParseCriteriaServiceType(svcType.String()); err == nil {
				criteria.Service = parsed
				detection.Service = &DetectionSource{
					Value:    string(parsed),
					Source:   "K8s server.service field",
					RawValue: svcType.String(),
				}
			}
		}

		// Extract service from K8s version string (only if not already detected)
		if detection.Service == nil {
			if version, ok := st.Data["version"]; ok {
				versionStr := version.String()
				var detectedService recipe.CriteriaServiceType
				switch {
				case strings.Contains(versionStr, "-eks-"):
					detectedService = recipe.CriteriaServiceEKS
				case strings.Contains(versionStr, "-gke"):
					detectedService = recipe.CriteriaServiceGKE
				case strings.Contains(versionStr, "-aks"):
					detectedService = recipe.CriteriaServiceAKS
				}
				if detectedService != "" {
					criteria.Service = detectedService
					detection.Service = &DetectionSource{
						Value:    string(detectedService),
						Source:   "K8s version string",
						RawValue: versionStr,
					}
				}
			}
		}
	}
}

// extractGPUCriteria extracts GPU/accelerator type from measurements.
func extractGPUCriteria(m *measurement.Measurement, criteria *recipe.Criteria, detection *CriteriaDetection) {
	for _, st := range m.Subtypes {
		if st.Name != "smi" && st.Name != "device" {
			continue
		}
		// Try "gpu.model" field (from nvidia-smi)
		if model, ok := st.Data["gpu.model"]; ok {
			if acc := detectAcceleratorFromModel(model.String()); acc != "" {
				criteria.Accelerator = acc
				detection.Accelerator = &DetectionSource{
					Value:    string(acc),
					Source:   "nvidia-smi gpu.model",
					RawValue: model.String(),
				}
			}
		}

		// Try plain "model" field (if not already detected)
		if detection.Accelerator == nil {
			if model, ok := st.Data["model"]; ok {
				if acc := detectAcceleratorFromModel(model.String()); acc != "" {
					criteria.Accelerator = acc
					detection.Accelerator = &DetectionSource{
						Value:    string(acc),
						Source:   "GPU model field",
						RawValue: model.String(),
					}
				}
			}
		}
	}
}

// extractOSCriteria extracts OS type from measurements.
func extractOSCriteria(m *measurement.Measurement, criteria *recipe.Criteria, detection *CriteriaDetection) {
	for _, st := range m.Subtypes {
		if st.Name != "release" {
			continue
		}
		if osID, ok := st.Data["ID"]; ok {
			if parsed, err := recipe.ParseCriteriaOSType(osID.String()); err == nil {
				criteria.OS = parsed
				detection.OS = &DetectionSource{
					Value:    string(parsed),
					Source:   "/etc/os-release ID",
					RawValue: osID.String(),
				}
			}
		}
	}
}

// applyCriteriaOverrides applies CLI flag overrides to criteria and tracks them in detection.
func applyCriteriaOverrides(cmd *cli.Command, criteria *recipe.Criteria, detection *CriteriaDetection) error {
	if s := cmd.String("service"); s != "" {
		parsed, err := recipe.ParseCriteriaServiceType(s)
		if err != nil {
			return err
		}
		criteria.Service = parsed
		detection.Service = &DetectionSource{
			Value:      string(parsed),
			Source:     "--service flag",
			Overridden: detection.Service != nil, // Was previously detected
		}
	}
	if s := cmd.String("accelerator"); s != "" {
		parsed, err := recipe.ParseCriteriaAcceleratorType(s)
		if err != nil {
			return err
		}
		criteria.Accelerator = parsed
		detection.Accelerator = &DetectionSource{
			Value:      string(parsed),
			Source:     "--accelerator flag",
			Overridden: detection.Accelerator != nil, // Was previously detected
		}
	}
	if s := cmd.String("intent"); s != "" {
		parsed, err := recipe.ParseCriteriaIntentType(s)
		if err != nil {
			return err
		}
		criteria.Intent = parsed
		detection.Intent = &DetectionSource{
			Value:      string(parsed),
			Source:     "--intent flag",
			Overridden: detection.Intent != nil, // Was previously detected
		}
	}
	if s := cmd.String("os"); s != "" {
		parsed, err := recipe.ParseCriteriaOSType(s)
		if err != nil {
			return err
		}
		criteria.OS = parsed
		detection.OS = &DetectionSource{
			Value:      string(parsed),
			Source:     "--os flag",
			Overridden: detection.OS != nil, // Was previously detected
		}
	}
	if n := cmd.Int("nodes"); n > 0 {
		criteria.Nodes = n
		detection.Nodes = &DetectionSource{
			Value:      fmt.Sprintf("%d", n),
			Source:     "--nodes flag",
			Overridden: detection.Nodes != nil, // Was previously detected
		}
	}
	return nil
}

// containsIgnoreCase checks if s contains substr (case-insensitive).
func containsIgnoreCase(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr ||
		len(s) > 0 && len(substr) > 0 &&
			(s[0]|0x20 == substr[0]|0x20) && containsIgnoreCase(s[1:], substr[1:]) ||
		len(s) > 0 && containsIgnoreCase(s[1:], substr))
}
