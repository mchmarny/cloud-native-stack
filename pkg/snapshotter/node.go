package snapshotter

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/NVIDIA/cloud-native-stack/pkg/collectors"
	"github.com/NVIDIA/cloud-native-stack/pkg/measurement"
	"github.com/NVIDIA/cloud-native-stack/pkg/node"
	"github.com/NVIDIA/cloud-native-stack/pkg/serializers"

	"golang.org/x/sync/errgroup"
)

type Snapshot struct {
	Kind         string                     `json:"kind,omitempty" yaml:"kind,omitempty"`
	APIVersion   string                     `json:"apiVersion,omitempty" yaml:"apiVersion,omitempty"`
	Metadata     map[string]string          `json:"metadata,omitempty" yaml:"metadata,omitempty"`
	Measurements []*measurement.Measurement `json:"measurements" yaml:"measurements"`
}

// NodeSnapshotter is a snapshotter that collects configuration from the current node.
type NodeSnapshotter struct {
	Version    string
	Factory    collectors.CollectorFactory
	Serializer serializers.Serializer
}

// Run collects configuration from the current node and outputs it to stdout.
// It implements the Snapshotter interface.
func (n *NodeSnapshotter) Run(ctx context.Context) error {
	if n.Factory == nil {
		n.Factory = collectors.NewDefaultCollectorFactory()
	}

	slog.Debug("starting node snapshot")

	// Pre-allocate with estimated capacity
	var mu sync.Mutex
	g, ctx := errgroup.WithContext(ctx)

	// Initialize snapshot structure
	snap := &Snapshot{
		Kind:         "Snapshot",
		APIVersion:   "snapshot.dgxc.io/v1",
		Metadata:     make(map[string]string),
		Measurements: make([]*measurement.Measurement, 0),
	}

	// Collect node metadata
	g.Go(func() error {
		slog.Debug("collecting node metadata")
		nd, err := node.Get(node.GetOptions{})
		if err != nil {
			slog.Error("failed to get node info", slog.String("error", err.Error()))
			return fmt.Errorf("failed to get node info: %w", err)
		}
		mu.Lock()
		snap.Metadata["snapshot-version"] = n.Version
		snap.Metadata["source-node"] = nd.Name
		snap.Metadata["snapshot-timestamp"] = time.Now().UTC().Format(time.RFC3339)
		mu.Unlock()
		slog.Debug("obtained node metadata", slog.String("name", nd.Name), slog.String("namespace", nd.Namespace))
		return nil
	})

	// Collect images
	g.Go(func() error {
		slog.Debug("collecting container images")
		ic := n.Factory.CreateImageCollector()
		images, err := ic.Collect(ctx)
		if err != nil {
			slog.Error("failed to collect container images", slog.String("error", err.Error()))
			return fmt.Errorf("failed to collect container images: %w", err)
		}
		mu.Lock()
		snap.Measurements = append(snap.Measurements, images)
		mu.Unlock()
		return nil
	})

	// Collect k8s resources
	g.Go(func() error {
		slog.Debug("collecting kubernetes resources")
		kc := n.Factory.CreateKubernetesCollector()
		k8sResources, err := kc.Collect(ctx)
		if err != nil {
			slog.Error("failed to collect kubernetes resources", slog.String("error", err.Error()))
			return fmt.Errorf("failed to collect kubernetes resources: %w", err)
		}
		mu.Lock()
		snap.Measurements = append(snap.Measurements, k8sResources)
		mu.Unlock()
		return nil
	})

	// Collect kernel modules
	g.Go(func() error {
		slog.Debug("collecting kernel modules")
		km := n.Factory.CreateKModCollector()
		kMod, err := km.Collect(ctx)
		if err != nil {
			slog.Error("failed to collect kmod", slog.String("error", err.Error()))
			return fmt.Errorf("failed to collect kMod info: %w", err)
		}
		mu.Lock()
		snap.Measurements = append(snap.Measurements, kMod)
		mu.Unlock()
		return nil
	})

	// Collect systemd
	g.Go(func() error {
		slog.Debug("collecting systemd services")
		sd := n.Factory.CreateSystemDCollector()
		systemd, err := sd.Collect(ctx)
		if err != nil {
			slog.Error("failed to collect systemd", slog.String("error", err.Error()))
			return fmt.Errorf("failed to collect systemd info: %w", err)
		}
		mu.Lock()
		snap.Measurements = append(snap.Measurements, systemd)
		mu.Unlock()
		return nil
	})

	// Collect grub
	g.Go(func() error {
		slog.Debug("collecting grub configuration")
		g := n.Factory.CreateGrubCollector()
		grub, err := g.Collect(ctx)
		if err != nil {
			slog.Error("failed to collect grub", slog.String("error", err.Error()))
			return fmt.Errorf("failed to collect grub info: %w", err)
		}
		mu.Lock()
		snap.Measurements = append(snap.Measurements, grub)
		mu.Unlock()
		return nil
	})

	// Collect sysctl
	g.Go(func() error {
		slog.Debug("collecting sysctl configuration")
		s := n.Factory.CreateSysctlCollector()
		sysctl, err := s.Collect(ctx)
		if err != nil {
			slog.Error("failed to collect sysctl", slog.String("error", err.Error()))
			return fmt.Errorf("failed to collect sysctl info: %w", err)
		}
		mu.Lock()
		snap.Measurements = append(snap.Measurements, sysctl)
		mu.Unlock()
		return nil
	})

	// Collect SMI
	g.Go(func() error {
		slog.Debug("collecting SMI configuration")
		smi := n.Factory.CreateSMICollector()
		smiConfigs, err := smi.Collect(ctx)
		if err != nil {
			slog.Error("failed to collect SMI", slog.String("error", err.Error()))
			return fmt.Errorf("failed to collect SMI info: %w", err)
		}
		mu.Lock()
		snap.Measurements = append(snap.Measurements, smiConfigs)
		mu.Unlock()
		return nil
	})

	// Wait for all collectors to complete
	if err := g.Wait(); err != nil {
		return err
	}

	slog.Debug("snapshot collection complete", slog.Int("total_configs", len(snap.Measurements)))

	// Serialize output
	if n.Serializer == nil {
		n.Serializer = serializers.NewStdoutWriter(serializers.FormatJSON)
	}

	if err := n.Serializer.Serialize(snap); err != nil {
		slog.Error("failed to serialize", slog.String("error", err.Error()))
		return fmt.Errorf("failed to serialize: %w", err)
	}

	return nil
}
