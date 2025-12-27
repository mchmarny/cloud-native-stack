package k8s

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/NVIDIA/cloud-native-stack/pkg/measurement"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (k *Collector) collectNode(ctx context.Context) (map[string]measurement.Reading, error) {
	// Check if context is canceled
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	// Get the current node name from environment
	nodeName := GetNodeName()
	if nodeName == "" {
		return nil, fmt.Errorf("node name not set in environment")
	}

	// Get node information from Kubernetes API
	node, err := k.ClientSet.CoreV1().Nodes().Get(ctx, nodeName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get node %q: %w", nodeName, err)
	}

	providerData := make(map[string]measurement.Reading)

	// Name
	providerData["source-node"] = measurement.Str(nodeName)

	// Provider
	providerID := node.Spec.ProviderID
	if providerID != "" {
		providerName := parseProvider(providerID)
		providerData["provider"] = measurement.Str(providerName)
		providerData["provider-id"] = measurement.Str(providerID)
	}

	// Node CRI-O
	status := node.Status
	if status.NodeInfo.ContainerRuntimeVersion != "" {
		providerData["container-runtime"] = measurement.Str(status.NodeInfo.ContainerRuntimeVersion)
	}

	if status.NodeInfo.KernelVersion != "" {
		providerData["kernel-version"] = measurement.Str(status.NodeInfo.KernelVersion)
	}

	if status.NodeInfo.OperatingSystem != "" {
		providerData["operating-system"] = measurement.Str(status.NodeInfo.OperatingSystem)
	}

	if status.NodeInfo.OSImage != "" {
		providerData["os-image"] = measurement.Str(status.NodeInfo.OSImage)
	}

	return providerData, nil
}

// parseProvider extracts the cloud provider name from a providerID string.
// Typical formats:
//   - aws:///us-west-2a/i-0123456789abcdef0 → "eks"
//   - gce://my-project/us-central1-a/gke-cluster-node → "gke"
//   - azure:///subscriptions/.../virtualMachines/... → "aks"
//   - oci://... → "oke"
//
// If the format is unrecognized, it returns the raw provider prefix.
func parseProvider(providerID string) string {
	if providerID == "" {
		slog.Warn("empty providerID string")
		return ""
	}

	// Split by "://" to get the provider prefix
	parts := strings.SplitN(providerID, "://", 2)
	if len(parts) < 1 {
		slog.Warn("invalid providerID format", slog.String("providerID", providerID))
		return ""
	}

	// Normalize provider names
	provider := strings.ToLower(strings.TrimSpace(parts[0]))

	switch provider {
	case "aws":
		return "eks"
	case "gce":
		return "gke"
	case "azure":
		return "aks"
	case "oci":
		return "oke"
	default:
		return provider
	}
}

// getNodeName retrieves the current node name from environment variables.
// It checks NODE_NAME first (typically set via Downward API), then falls back
// to KUBERNETES_NODE_NAME, and finally HOSTNAME as a last resort.
func GetNodeName() string {
	// Preferred: NODE_NAME set via Downward API
	if nodeName := os.Getenv("NODE_NAME"); nodeName != "" {
		return nodeName
	}

	// Alternative: KUBERNETES_NODE_NAME
	if nodeName := os.Getenv("KUBERNETES_NODE_NAME"); nodeName != "" {
		return nodeName
	}

	// Last resort: HOSTNAME (may be pod name, not node name)
	if hostname := os.Getenv("HOSTNAME"); hostname != "" {
		return hostname
	}

	return ""
}
