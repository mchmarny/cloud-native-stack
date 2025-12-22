package collectors

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/NVIDIA/cloud-native-stack/pkg/client"
	"github.com/NVIDIA/cloud-native-stack/pkg/measurement"
)

// KubernetesCollector collects information about the Kubernetes cluster.
type KubernetesCollector struct {
}

// Collect retrieves Kubernetes cluster version information from the API server.
// This provides cluster version details for comparison across environments.
func (k *KubernetesCollector) Collect(ctx context.Context) (*measurement.Measurement, error) {
	// Check if context is canceled
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	k8sClient, _, err := client.GetKubeClient("")
	if err != nil {
		return nil, fmt.Errorf("failed to get kubernetes client: %w", err)
	}

	serverVersion, err := k8sClient.Discovery().ServerVersion()
	if err != nil {
		return nil, fmt.Errorf("failed to get kubernetes version: %w", err)
	}

	versionInfo := map[string]measurement.Reading{
		"version":      measurement.Str(serverVersion.GitVersion),
		"platform":     measurement.Str(serverVersion.Platform),
		"buildDate":    measurement.Str(serverVersion.BuildDate),
		"goVersion":    measurement.Str(serverVersion.GoVersion),
		"gitCommit":    measurement.Str(serverVersion.GitCommit),
		"gitTreeState": measurement.Str(serverVersion.GitTreeState),
	}

	slog.Debug("collected kubernetes version", slog.String("version", serverVersion.GitVersion))

	res := &measurement.Measurement{
		Type: measurement.TypeK8s,
		Subtypes: []measurement.Subtype{
			{
				Data: versionInfo,
			},
		},
	}

	return res, nil
}
