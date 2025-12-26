package recommender

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/NVIDIA/cloud-native-stack/pkg/measurement"
	"github.com/NVIDIA/cloud-native-stack/pkg/recipe"
	"github.com/NVIDIA/cloud-native-stack/pkg/snapshotter"
	"github.com/NVIDIA/cloud-native-stack/pkg/version"
)

// ConfigurationRecommender is a recommender that suggests configuration changes.
// Implements the Recommender interface.
type ConfigurationRecommender struct {
	Version string
}

// Option is a functional option for configuring the ConfigurationRecommender
type Option func(*ConfigurationRecommender)

func WithVersion(version string) Option {
	return func(r *ConfigurationRecommender) {
		r.Version = version
	}
}

// New creates a new ConfigurationRecommender with the provided options.
func New(opts ...Option) *ConfigurationRecommender {
	s := &ConfigurationRecommender{}

	// Apply options
	for _, opt := range opts {
		opt(s)
	}

	return s
}

// Recommend generates configuration recommendations based on the provided snapshot.
func (r *ConfigurationRecommender) Recommend(ctx context.Context, intent recipe.IntentType, snap *snapshotter.Snapshot) (*Recommendation, error) {
	if snap == nil {
		return nil, fmt.Errorf("snapshot cannot be nil")
	}

	if len(snap.Measurements) == 0 {
		return nil, fmt.Errorf("snapshot contains no measurements")
	}

	if !intent.IsValid() {
		return nil, fmt.Errorf("invalid intent type: %q", intent)
	}

	// Check for context cancellation
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	// Track overall snapshot collection duration
	start := time.Now()
	defer func() {
		recommendGenerateDuration.Observe(time.Since(start).Seconds())
	}()

	query, err := QueryFromSnapshot(intent, snap)
	if err != nil {
		return nil, fmt.Errorf("failed to extract query from snapshot: %w", err)
	}

	slog.Debug("extracted recipe query from snapshot",
		"os", query.Os.String(),
		"os_version", query.OsVersion.String(),
		"kernel", query.Kernel.String(),
		"service", query.Service.String(),
		"k8s", query.K8s.String(),
		"gpu", query.GPU.String(),
		"intent", intent.String(),
		"include_context", query.IncludeContext,
	)

	// TODO: Implement actual recommendation logic

	recommendGenerateTotal.WithLabelValues("success").Inc()

	// Placeholder implementation
	return nil, fmt.Errorf("configuration recommendations not implemented yet")
}

// QueryFromSnapshot extracts a recipe.Query from the provided snapshot.
// This function analyzes the snapshot data to determine the best source of
// all of the fields required to build a recipe.Query.
func QueryFromSnapshot(intent recipe.IntentType, snap *snapshotter.Snapshot) (*recipe.Query, error) {
	if snap == nil {
		return nil, fmt.Errorf("snapshot cannot be nil")
	}

	if len(snap.Measurements) == 0 {
		return nil, fmt.Errorf("snapshot contains no measurements")
	}

	// Create a Query with the IncludeContext flag set by default
	q := &recipe.Query{
		Intent:         intent,
		IncludeContext: true,
	}

	// Analyze snapshot measurements to populate the Query fields
	for _, m := range snap.Measurements {
		if m == nil {
			continue
		}

		if parser, exists := subtypeQueryParsers[m.Type]; exists {
			if err := parser(m.Subtypes, q); err != nil {
				slog.Error("error parsing subtypes for measurement type",
					slog.String("type", string(m.Type)),
					slog.Any("error", err),
				)
			}
			continue
		}
	}

	// Placeholder implementation
	return q, nil
}

var (
	// subtypeQueryParsers maps measurement types to their respective subtype parsers
	// used in QueryFromSnapshot.
	subtypeQueryParsers = map[measurement.Type]SubtypeQueryParser{
		measurement.TypeOS:  parseOSSubtypes,
		measurement.TypeK8s: parseK8sSubtypes,
		measurement.TypeGPU: parseGPUSubtypes,
	}
)

// SubtypeQueryParser is a function that parses subtypes from measurements
type SubtypeQueryParser func(data []measurement.Subtype, q *recipe.Query) error

// parseOSSubtypes parses OS-related subtypes to populate the Os, OsVersion, and Kernel fields in the recipe.Query.
// It looks for the OS family, OS version, and kernel version in the measurement subtypes.
// If found, it sets the corresponding fields in the recipe.Query.
func parseOSSubtypes(subtypes []measurement.Subtype, q *recipe.Query) error {
	// Parse kernel version
	reading := walkSubtypesForReading("sysctl", "/proc/sys/kernel/osrelease", subtypes)
	if reading != nil {
		v, err := version.ParseVersion(reading.String())
		if err != nil {
			return fmt.Errorf("failed to parse kernel version from reading %q: %w", reading.String(), err)
		}
		q.Kernel = &v
	}

	// Parse OS family
	reading = walkSubtypesForReading("release", "ID", subtypes)
	if reading != nil {
		os := recipe.OsFamily(reading.String())
		if !os.IsValid() {
			return fmt.Errorf("invalid OS family parsed from reading %q", reading.String())
		}
		q.Os = os
	}

	// Parse OS version
	reading = walkSubtypesForReading("release", "VERSION_ID", subtypes)
	if reading != nil {
		v, err := version.ParseVersion(reading.String())
		if err != nil {
			return fmt.Errorf("failed to parse OS version from reading %q: %w", reading.String(), err)
		}
		q.OsVersion = &v
	}

	return nil
}

// parseK8sSubtypes parses Kubernetes-related subtypes to populate the K8s and Service fields in the recipe.Query.
// It looks for the Kubernetes version and service type in the measurement subtypes.
// If found, it sets the corresponding fields in the recipe.Query.
func parseK8sSubtypes(subtypes []measurement.Subtype, q *recipe.Query) error {
	// Parse Kubernetes version
	reading := walkSubtypesForReading("server", "version", subtypes)
	if reading != nil {
		v, err := version.ParseVersion(reading.String())
		if err != nil {
			return fmt.Errorf("failed to parse Kubernetes version from reading %q: %w", reading.String(), err)
		}
		q.K8s = &v
	}

	// Parse Service type
	reading = walkSubtypesForReading("provider", "service", subtypes)
	if reading != nil {
		service := recipe.ServiceType(reading.String())
		if !service.IsValid() {
			return fmt.Errorf("invalid Service type parsed from reading %q", reading.String())
		}
		q.Service = service
	}

	return nil
}

// parseGPUSubtypes parses GPU-related subtypes to populate the GPU field in the recipe.Query.
// It looks for the GPU model in the measurement subtypes and maps it to a known GPUType.
// If the GPU model is unrecognized, it uses the raw value as the GPUType.
func parseGPUSubtypes(subtypes []measurement.Subtype, q *recipe.Query) error {
	// Parse GPU type
	reading := walkSubtypesForReading("smi", "gpu.model", subtypes)
	if reading != nil {
		gpuVal := reading.String()
		// Normalize GPU model to GPUType
		var gpuType recipe.GPUType
		switch {
		case strings.Contains(gpuVal, "GB200"):
			gpuType = recipe.GPUB200
		case strings.Contains(gpuVal, "H100"):
			gpuType = recipe.GPUH100
		default:
			gpuType = recipe.GPUType(gpuVal) // Use raw value for unknown models
		}

		if !gpuType.IsValid() {
			return fmt.Errorf("invalid GPU type parsed from reading %q", reading.String())
		}

		q.GPU = gpuType
	}

	return nil
}

// walkSubtypesForReading searches through the provided subtypes
// for a measurement reading with the specified name and key. If found, it
// returns the reading; otherwise, it returns nil.
func walkSubtypesForReading(name, key string, data []measurement.Subtype) measurement.Reading {
	for _, st := range data {
		if st.Name != name {
			continue
		}
		if reading, exists := st.Data[key]; exists {
			return reading
		}
	}
	return nil
}
