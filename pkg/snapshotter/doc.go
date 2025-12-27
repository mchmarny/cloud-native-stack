// Package snapshotter captures comprehensive system configuration snapshots.
//
// # Overview
//
// The snapshotter package orchestrates parallel collection of system measurements
// from multiple sources (Kubernetes, GPU, OS, SystemD) and produces structured
// snapshots that can be serialized for analysis, auditing, or recommendation generation.
//
// # Core Types
//
// Snapshotter: Interface for snapshot collection
//
//	type Snapshotter interface {
//	    Measure(ctx context.Context) error
//	}
//
// NodeSnapshotter: Production implementation that collects from the current node
//
//	type NodeSnapshotter struct {
//	    Version    string               // Snapshotter version
//	    Factory    collector.Factory    // Collector factory (optional)
//	    Serializer serializer.Serializer // Output serializer (optional)
//	}
//
// Snapshot: Captured configuration data
//
//	type Snapshot struct {
//	    Header                            // API version, kind, metadata
//	    Measurements []*measurement.Measurement // Collected data
//	}
//
// # Usage
//
// Basic snapshot with defaults (stdout YAML):
//
//	snapshotter := &snapshotter.NodeSnapshotter{
//	    Version: "v1.0.0",
//	}
//
//	ctx := context.Background()
//	if err := snapshotter.Measure(ctx); err != nil {
//	    log.Fatalf("snapshot failed: %v", err)
//	}
//
// Custom collector factory:
//
//	factory := collector.NewDefaultFactory(
//	    collector.WithSystemDServices([]string{"containerd.service"}),
//	)
//
//	snapshotter := &snapshotter.NodeSnapshotter{
//	    Version: "v1.0.0",
//	    Factory: factory,
//	}
//
//	if err := snapshotter.Measure(context.Background()); err != nil {
//	    log.Fatal(err)
//	}
//
// Custom output serializer:
//
//	serializer, err := serializer.NewFileSerializer("snapshot.json")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer serializer.Close()
//
//	snapshotter := &snapshotter.NodeSnapshotter{
//	    Version:    "v1.0.0",
//	    Serializer: serializer,
//	}
//
//	if err := snapshotter.Measure(context.Background()); err != nil {
//	    log.Fatal(err)
//	}
//
// With timeout:
//
//	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
//	defer cancel()
//
//	snapshotter := &snapshotter.NodeSnapshotter{Version: "v1.0.0"}
//	if err := snapshotter.Measure(ctx); err != nil {
//	    log.Fatal(err)
//	}
//
// # Snapshot Structure
//
// Snapshots contain a header and measurements:
//
//	apiVersion: snapshot.dgxc.io/v1
//	kind: Snapshot
//	metadata:
//	  snapshot-version: v1.0.0
//	  source-node: node-1
//	  created: 2025-01-15T10:30:00Z
//	measurements:
//	  - type: K8s
//	    subtypes:
//	      - subtype: server
//	        data:
//	          version: 1.33.5
//	          platform: linux/amd64
//	      - subtype: node
//	        data:
//	          provider: eks
//	          kernel-version: 6.8.0
//	  - type: GPU
//	    subtypes:
//	      - subtype: device
//	        data:
//	          driver: 570.158.01
//	          model: H100
//
// # Parallel Collection
//
// NodeSnapshotter runs all collectors concurrently using errgroup:
//  1. Metadata collection (node name, version)
//  2. Kubernetes resources (cluster config, policies)
//  3. SystemD services (containerd, kubelet)
//  4. OS configuration (grub, sysctl, modules)
//  5. GPU hardware (driver, model, settings)
//
// If any collector fails, all are canceled and an error is returned.
//
// # Node Name Detection
//
// Node name is determined with fallback priority:
//  1. NODE_NAME environment variable
//  2. KUBERNETES_NODE_NAME environment variable
//  3. HOSTNAME environment variable
//
// This ensures correct node identification in various deployment scenarios.
//
// # Error Handling
//
// Measure() returns an error when:
//   - Any collector fails
//   - Context is canceled or times out
//   - Serialization fails
//
// Partial data is never returned - snapshots are all-or-nothing.
//
// # Observability
//
// The snapshotter exports Prometheus metrics:
//   - snapshot_collection_duration_seconds: Total time to collect snapshot
//   - snapshot_collector_duration_seconds{collector}: Per-collector timing
//
// Structured logs are emitted for:
//   - Snapshot start
//   - Collector progress
//   - Errors and failures
//
// # Resource Requirements
//
// Collectors may require:
//   - Kubernetes API access (in-cluster config or kubeconfig)
//   - NVIDIA GPU and nvidia-smi binary
//   - systemd and systemctl binary
//   - Read access to /proc, /sys, /etc
//
// Failures due to missing resources are reported as errors.
//
// # Integration
//
// The snapshotter is invoked by:
//   - pkg/cli - snapshot command
//   - Kubernetes Job - eidos-agent deployment
//
// It depends on:
//   - pkg/collector - Data collection implementations
//   - pkg/serializer - Output formatting
//   - pkg/measurement - Data structures
//
// Snapshots are consumed by:
//   - pkg/recommender - Recommendation generation
//   - External analysis tools
//   - Auditing and compliance systems
package snapshotter
