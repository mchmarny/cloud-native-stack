package k8s

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/NVIDIA/cloud-native-stack/pkg/k8s/client"
	"github.com/NVIDIA/cloud-native-stack/pkg/measurement"
	"k8s.io/client-go/kubernetes"
)

// Collector collects information about the Kubernetes cluster.
type Collector struct {
	Clientset kubernetes.Interface
}

// Collect retrieves Kubernetes cluster version information from the API server.
// This provides cluster version details for comparison across environments.
func (k *Collector) Collect(ctx context.Context) (*measurement.Measurement, error) {
	// Check if context is canceled
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	k8sClient, err := k.getClient()
	if err != nil {
		return nil, err
	}

	serverVersion, err := k8sClient.Discovery().ServerVersion()
	if err != nil {
		return nil, fmt.Errorf("failed to get kubernetes version: %w", err)
	}

	versionInfo := map[string]measurement.Reading{
		"version":   measurement.Str(serverVersion.GitVersion),
		"platform":  measurement.Str(serverVersion.Platform),
		"goVersion": measurement.Str(serverVersion.GoVersion),
	}

	slog.Debug("collected kubernetes version", slog.String("version", serverVersion.GitVersion))

	res := &measurement.Measurement{
		Type: measurement.TypeK8s,
		Subtypes: []measurement.Subtype{
			{
				Data: versionInfo, // no need for filtering, all fields explicitly collected
			},
		},
	}

	return res, nil
}

func (k *Collector) getClient() (kubernetes.Interface, error) {
	if k.Clientset != nil {
		return k.Clientset, nil
	}
	k8sClient, _, err := client.GetKubeClient("")
	if err != nil {
		return nil, fmt.Errorf("failed to get kubernetes client: %w", err)
	}
	return k8sClient, nil
}
