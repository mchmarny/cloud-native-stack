package collectors

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os/exec"
)

// HelmCollector collects information about Helm releases in the cluster.
type HelmCollector struct {
}

// HelmType is the type identifier for Helm measurements.
const HelmType string = "Helm"

// Collect retrieves all Helm releases across all namespaces.
// This provides a reliable snapshot of installed charts for cluster comparison.
func (h *HelmCollector) Collect(ctx context.Context) ([]Measurement, error) {
	// Check if context is canceled
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	releases, err := h.collectHelmReleases(ctx)
	if err != nil {
		return nil, err
	}

	res := []Measurement{
		{
			Type: HelmType,
			Data: releases,
		},
	}

	return res, nil
}

// collectHelmReleases retrieves all Helm releases across all namespaces.
func (h *HelmCollector) collectHelmReleases(ctx context.Context) (map[string]any, error) {
	// Check if helm is available
	helmPath, err := exec.LookPath("helm")
	if err != nil {
		return nil, fmt.Errorf("helm not found in PATH: %w", err)
	}

	// Run helm list -A -o json
	cmd := exec.CommandContext(ctx, helmPath, "list", "-A", "--output", "json")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to execute helm list: %w", err)
	}

	// Parse JSON output
	var releases []struct {
		Name       string `json:"name"`
		Namespace  string `json:"namespace"`
		Chart      string `json:"chart"`
		AppVersion string `json:"app_version"`
		Status     string `json:"status"`
	}

	if err := json.Unmarshal(output, &releases); err != nil {
		return nil, fmt.Errorf("failed to parse helm releases: %w", err)
	}

	result := make(map[string]any)
	for _, rel := range releases {
		key := fmt.Sprintf("%s/%s", rel.Namespace, rel.Name)
		result[key] = map[string]string{
			"chart":      rel.Chart,
			"appVersion": rel.AppVersion,
			"status":     rel.Status,
		}
	}

	slog.Debug("collected helm releases", slog.Int("count", len(result)))
	return result, nil
}
