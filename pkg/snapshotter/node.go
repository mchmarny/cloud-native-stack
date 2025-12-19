package snapshotter

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/NVIDIA/cloud-native-stack/pkg/collectors"
	"github.com/NVIDIA/cloud-native-stack/pkg/node"
	"github.com/NVIDIA/cloud-native-stack/pkg/serializers"

	"golang.org/x/sync/errgroup"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Snapshot struct {
	metav1.TypeMeta `json:",inline"`
	// Standard object's metadata.
	// More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	Measurements []collectors.Measurement `json:"measurements" yaml:"measurements"`
}

// NodeSnapshotter is a snapshotter that collects configuration from the current node.
type NodeSnapshotter struct {
	Factory    collectors.CollectorFactory
	Serializer serializers.Serializer
	Logger     *slog.Logger
}

// Run collects configuration from the current node and outputs it to stdout.
// It implements the Snapshotter interface.
func (n *NodeSnapshotter) Run(ctx context.Context) error {
	if n.Logger == nil {
		n.Logger = slog.Default()
	}
	if n.Factory == nil {
		n.Factory = collectors.NewDefaultCollectorFactory()
	}

	n.Logger.Info("starting node snapshot")

	// Pre-allocate with estimated capacity
	var mu sync.Mutex
	g, ctx := errgroup.WithContext(ctx)

	// Initialize snapshot structure
	snap := &Snapshot{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Snapshot",
			APIVersion: "snapshot.dgxc.io/v1",
		},
		Measurements: make([]collectors.Measurement, 0),
	}

	// Collect node metadata
	g.Go(func() error {
		n.Logger.Debug("collecting node metadata")
		nd, err := node.Get(node.GetOptions{})
		if err != nil {
			n.Logger.Error("failed to get node info", slog.String("error", err.Error()))
			return fmt.Errorf("failed to get node info: %w", err)
		}
		mu.Lock()
		snap.Name = nd.Name
		snap.Namespace = nd.Namespace
		// snap.Labels = nd.Labels // Noisy but uncomment if labels are needed
		snap.Annotations = map[string]string{
			"snapshot-time": time.Now().UTC().Format(time.RFC1123Z),
		}
		mu.Unlock()
		n.Logger.Debug("obtained node metadata", slog.String("name", nd.Name), slog.String("namespace", nd.Namespace))
		return nil
	})

	// Collect kernel modules concurrently
	g.Go(func() error {
		n.Logger.Debug("collecting kernel modules")
		km := n.Factory.CreateKModCollector()
		kMod, err := km.Collect(ctx)
		if err != nil {
			n.Logger.Error("failed to collect kmod", slog.String("error", err.Error()))
			return fmt.Errorf("failed to collect kMod info: %w", err)
		}
		mu.Lock()
		snap.Measurements = append(snap.Measurements, kMod...)
		mu.Unlock()
		n.Logger.Debug("collected kernel modules", slog.Int("count", len(kMod)))
		return nil
	})

	// Collect systemd concurrently
	g.Go(func() error {
		n.Logger.Debug("collecting systemd services")
		sd := n.Factory.CreateSystemDCollector()
		systemd, err := sd.Collect(ctx)
		if err != nil {
			n.Logger.Error("failed to collect systemd", slog.String("error", err.Error()))
			return fmt.Errorf("failed to collect systemd info: %w", err)
		}
		mu.Lock()
		snap.Measurements = append(snap.Measurements, systemd...)
		mu.Unlock()
		n.Logger.Debug("collected systemd services", slog.Int("count", len(systemd)))
		return nil
	})

	// Collect grub concurrently
	g.Go(func() error {
		n.Logger.Debug("collecting grub configuration")
		g := n.Factory.CreateGrubCollector()
		grub, err := g.Collect(ctx)
		if err != nil {
			n.Logger.Error("failed to collect grub", slog.String("error", err.Error()))
			return fmt.Errorf("failed to collect grub info: %w", err)
		}
		mu.Lock()
		snap.Measurements = append(snap.Measurements, grub...)
		mu.Unlock()
		n.Logger.Debug("collected grub parameters", slog.Int("count", len(grub)))
		return nil
	})

	// Collect sysctl concurrently
	g.Go(func() error {
		n.Logger.Debug("collecting sysctl configuration")
		s := n.Factory.CreateSysctlCollector()
		sysctl, err := s.Collect(ctx)
		if err != nil {
			n.Logger.Error("failed to collect sysctl", slog.String("error", err.Error()))
			return fmt.Errorf("failed to collect sysctl info: %w", err)
		}
		mu.Lock()
		snap.Measurements = append(snap.Measurements, sysctl...)
		mu.Unlock()
		n.Logger.Debug("collected sysctl parameters", slog.Int("count", len(sysctl)))
		return nil
	})

	// Wait for all collectors to complete
	if err := g.Wait(); err != nil {
		return err
	}

	n.Logger.Info("snapshot collection complete", slog.Int("total_configs", len(snap.Measurements)))

	// Serialize output
	if n.Serializer == nil {
		n.Serializer = serializers.NewWriter(serializers.FormatJSON, nil)
	}

	if err := n.Serializer.Serialize(snap); err != nil {
		n.Logger.Error("failed to serialize", slog.String("error", err.Error()))
		return fmt.Errorf("failed to serialize: %w", err)
	}

	return nil
}
