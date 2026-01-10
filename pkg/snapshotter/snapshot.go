package snapshotter

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/NVIDIA/cloud-native-stack/pkg/collector"
	"github.com/NVIDIA/cloud-native-stack/pkg/collector/k8s"
	"github.com/NVIDIA/cloud-native-stack/pkg/measurement"
	"github.com/NVIDIA/cloud-native-stack/pkg/recipe/header"
	"github.com/NVIDIA/cloud-native-stack/pkg/serializer"

	"golang.org/x/sync/errgroup"
)

// NodeSnapshotter collects system configuration measurements from the current node.
// It coordinates multiple collectors in parallel to gather data about Kubernetes,
// GPU hardware, OS configuration, and systemd services, then serializes the results.
// If AgentConfig is provided with Enabled=true, it deploys a Kubernetes Job instead.
type NodeSnapshotter struct {
	// Version is the snapshotter version.
	Version string

	// Factory is the collector factory to use. If nil, the default factory is used.
	Factory collector.Factory

	// Serializer is the serializer to use for output. If nil, a default stdout JSON serializer is used.
	Serializer serializer.Serializer

	// AgentConfig contains configuration for agent deployment mode. If nil or Enabled=false, runs locally.
	AgentConfig *AgentConfig
}

// Measure collects configuration measurements and serializes the snapshot.
// If AgentConfig is enabled, it deploys a Kubernetes Job to capture the snapshot.
// Otherwise, it runs collectors locally in parallel using errgroup.
// If any collector fails, the entire operation returns an error.
// The resulting snapshot is serialized using the configured Serializer.
func (n *NodeSnapshotter) Measure(ctx context.Context) error {
	// Check if agent deployment is requested
	if n.AgentConfig != nil && n.AgentConfig.Enabled {
		return n.measureWithAgent(ctx)
	}

	// Local measurement mode
	return n.measure(ctx)
}

// measure collects configuration measurements from the current node.
func (n *NodeSnapshotter) measure(ctx context.Context) error {
	if n.Factory == nil {
		n.Factory = collector.NewDefaultFactory()
	}

	slog.Debug("starting node snapshot")

	// Track overall snapshot collection duration
	start := time.Now()
	defer func() {
		snapshotCollectionDuration.Observe(time.Since(start).Seconds())
	}()

	// Pre-allocate with estimated capacity
	var mu sync.Mutex

	// Using the gctx for errgroup goroutines and keeping original ctx for serialization.
	// The errgroup context is canceled when any goroutine returns an error or
	// when all goroutines complete, which causes issues with subsequent
	// operations that use rate-limited K8s clients.
	g, gctx := errgroup.WithContext(ctx)

	// Initialize snapshot structure
	snap := NewSnapshot()
	// Pre-allocate measurements slice with capacity for 5 collectors
	snap.Measurements = make([]*measurement.Measurement, 0, 5)

	// Collect metadata
	g.Go(func() error {
		collectorStart := time.Now()
		defer func() {
			snapshotCollectorDuration.WithLabelValues("metadata").Observe(time.Since(collectorStart).Seconds())
		}()
		nodeName := k8s.GetNodeName()
		mu.Lock()
		snap.Init(header.KindSnapshot, FullAPIVersion, n.Version)
		snap.Metadata["source-node"] = nodeName
		mu.Unlock()
		slog.Debug("obtained node metadata", slog.String("name", nodeName), slog.String("version", n.Version))
		return nil
	})

	// Collect Kubernetes configuration
	g.Go(func() error {
		collectorStart := time.Now()
		defer func() {
			snapshotCollectorDuration.WithLabelValues("k8s").Observe(time.Since(collectorStart).Seconds())
		}()
		slog.Debug("collecting kubernetes resources")
		kc := n.Factory.CreateKubernetesCollector()
		k8sResources, err := kc.Collect(gctx)
		if err != nil {
			slog.Error("failed to collect kubernetes resources", slog.String("error", err.Error()))
			return fmt.Errorf("failed to collect kubernetes resources: %w", err)
		}
		mu.Lock()
		snap.Measurements = append(snap.Measurements, k8sResources)
		mu.Unlock()
		return nil
	})

	// Collect SystemD services
	g.Go(func() error {
		collectorStart := time.Now()
		defer func() {
			snapshotCollectorDuration.WithLabelValues("systemd").Observe(time.Since(collectorStart).Seconds())
		}()
		slog.Debug("collecting systemd services")
		sd := n.Factory.CreateSystemDCollector()
		systemd, err := sd.Collect(gctx)
		if err != nil {
			slog.Error("failed to collect systemd", slog.String("error", err.Error()))
			return fmt.Errorf("failed to collect systemd info: %w", err)
		}
		mu.Lock()
		snap.Measurements = append(snap.Measurements, systemd)
		mu.Unlock()
		return nil
	})

	// Collect OS
	g.Go(func() error {
		collectorStart := time.Now()
		defer func() {
			snapshotCollectorDuration.WithLabelValues("os").Observe(time.Since(collectorStart).Seconds())
		}()
		slog.Debug("collecting OS configuration")
		oc := n.Factory.CreateOSCollector()
		grub, err := oc.Collect(gctx)
		if err != nil {
			slog.Error("failed to collect OS", slog.String("error", err.Error()))
			return fmt.Errorf("failed to collect OS info: %w", err)
		}
		mu.Lock()
		snap.Measurements = append(snap.Measurements, grub)
		mu.Unlock()
		return nil
	})

	// Collect GPU
	g.Go(func() error {
		collectorStart := time.Now()
		defer func() {
			snapshotCollectorDuration.WithLabelValues("gpu").Observe(time.Since(collectorStart).Seconds())
		}()
		slog.Debug("collecting GPU configuration")
		smi := n.Factory.CreateGPUCollector()
		smiConfigs, err := smi.Collect(gctx)
		if err != nil {
			slog.Error("failed to collect GPU", slog.String("error", err.Error()))
			return fmt.Errorf("failed to collect SMI info: %w", err)
		}
		mu.Lock()
		snap.Measurements = append(snap.Measurements, smiConfigs)
		mu.Unlock()
		return nil
	})

	// Wait for all collectors to complete
	if err := g.Wait(); err != nil {
		snapshotCollectionTotal.WithLabelValues("error").Inc()
		return err
	}

	snapshotCollectionTotal.WithLabelValues("success").Inc()
	snapshotMeasurementCount.Set(float64(len(snap.Measurements)))

	slog.Debug("snapshot collection complete", slog.Int("total_configs", len(snap.Measurements)))

	// Serialize output
	if n.Serializer == nil {
		n.Serializer = serializer.NewStdoutWriter(serializer.FormatJSON)
	}

	if err := n.Serializer.Serialize(ctx, snap); err != nil {
		slog.Error("failed to serialize", slog.String("error", err.Error()))
		return fmt.Errorf("failed to serialize: %w", err)
	}

	return nil
}
