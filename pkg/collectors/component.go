package collectors

import (
	"context"
	"log/slog"
)

// ComponentCollector combines Helm, Kubernetes, and Image collectors.
// This collector aggregates information from multiple sources for cluster comparison.
type ComponentCollector struct {
	helmCollector       *HelmCollector
	kubernetesCollector *KubernetesCollector
	imageCollector      *ImageCollector
}

// Collect retrieves component information from Helm releases, Kubernetes API, and container images.
// Returns measurements from all three sources, with warnings logged for any failures.
func (c *ComponentCollector) Collect(ctx context.Context) ([]Measurement, error) {
	// Check if context is canceled
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	// Initialize collectors if needed
	if c.helmCollector == nil {
		c.helmCollector = &HelmCollector{}
	}
	if c.kubernetesCollector == nil {
		c.kubernetesCollector = &KubernetesCollector{}
	}
	if c.imageCollector == nil {
		c.imageCollector = &ImageCollector{}
	}

	var allMeasurements []Measurement

	// Collect Helm releases
	helmMeasurements, err := c.helmCollector.Collect(ctx)
	if err != nil {
		slog.Warn("failed to collect helm releases", slog.String("error", err.Error()))
	} else {
		allMeasurements = append(allMeasurements, helmMeasurements...)
	}

	// Collect Kubernetes version
	k8sMeasurements, err := c.kubernetesCollector.Collect(ctx)
	if err != nil {
		slog.Warn("failed to collect kubernetes version", slog.String("error", err.Error()))
	} else {
		allMeasurements = append(allMeasurements, k8sMeasurements...)
	}

	// Collect container images
	imageMeasurements, err := c.imageCollector.Collect(ctx)
	if err != nil {
		slog.Warn("failed to collect container images", slog.String("error", err.Error()))
	} else {
		allMeasurements = append(allMeasurements, imageMeasurements...)
	}

	return allMeasurements, nil
}
