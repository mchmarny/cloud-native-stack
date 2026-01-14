package k8s

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/NVIDIA/cloud-native-stack/pkg/k8s/client"
	"github.com/NVIDIA/cloud-native-stack/pkg/measurement"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// Collector collects information about the Kubernetes cluster.
type Collector struct {
	ClientSet  kubernetes.Interface
	RestConfig *rest.Config
}

// Collect retrieves Kubernetes cluster version information from the API server.
// This provides cluster version details for comparison across environments.
func (k *Collector) Collect(ctx context.Context) (*measurement.Measurement, error) {
	slog.Info("collecting Kubernetes cluster information")

	// Check if context is canceled
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	if err := k.getClient(); err != nil {
		return nil, err
	}
	// Cluster Version
	versions, err := k.collectServer(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to collect server version: %w", err)
	}

	// Cluster Images
	images, err := k.collectContainerImages(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to collect container images: %w", err)
	}

	// Cluster Policies
	policies, err := k.collectClusterPolicies(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to collect cluster policies: %w", err)
	}

	// Node
	node, err := k.collectNode(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to collect node: %w", err)
	}

	// Build measurement using builder pattern
	res := measurement.NewMeasurement(measurement.TypeK8s).
		WithSubtypeBuilder(
			measurement.NewSubtypeBuilder("server").Set(measurement.KeyVersion, versions[measurement.KeyVersion]).
				Set("platform", versions["platform"]).
				Set("goVersion", versions["goVersion"]),
		).
		WithSubtype(measurement.Subtype{Name: "image", Data: images}).
		WithSubtype(measurement.Subtype{Name: "policy", Data: policies}).
		WithSubtype(measurement.Subtype{Name: "node", Data: node}).
		Build()

	return res, nil
}

func (k *Collector) getClient() error {
	if k.ClientSet != nil && k.RestConfig != nil {
		return nil
	}
	var err error
	k.ClientSet, k.RestConfig, err = client.GetKubeClient()
	if err != nil {
		return fmt.Errorf("failed to get kubernetes client: %w", err)
	}
	return nil
}
