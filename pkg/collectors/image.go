package collectors

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/NVIDIA/cloud-native-stack/pkg/client"
	"github.com/NVIDIA/cloud-native-stack/pkg/measurement"

	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// ImageCollector collects information about container images running in the cluster.
type ImageCollector struct {
}

// Collect retrieves unique container images from all pods in the cluster.
// This provides a snapshot of deployed images for cluster comparison.
func (i *ImageCollector) Collect(ctx context.Context) (*measurement.Measurement, error) {
	// Check if context is canceled
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	k8sClient, _, err := client.GetKubeClient("")
	if err != nil {
		return nil, fmt.Errorf("failed to get kubernetes client: %w", err)
	}

	images, err := i.collectContainerImages(ctx, k8sClient)
	if err != nil {
		return nil, err
	}

	res := &measurement.Measurement{
		Type: measurement.TypeImage,
		Subtypes: []measurement.Subtype{
			{
				Data: images,
			},
		},
	}

	return res, nil
}

// collectContainerImages extracts unique container images from all pods.
func (i *ImageCollector) collectContainerImages(ctx context.Context, k8sClient *kubernetes.Clientset) (map[string]measurement.Reading, error) {
	pods, err := k8sClient.CoreV1().Pods("").List(ctx, v1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list pods: %w", err)
	}

	images := make(map[string]measurement.Reading)
	for _, pod := range pods.Items {
		// Check for context cancellation
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		// Extract images from all containers (init, regular, ephemeral)
		allContainers := append([]corev1.Container{}, pod.Spec.Containers...)
		allContainers = append(allContainers, pod.Spec.InitContainers...)

		for _, container := range allContainers {
			imageName, imageTag := parseImageReference(container.Image)
			if imageName == "" {
				continue
			}

			// Store with image name as key
			if _, exists := images[imageName]; !exists {
				images[imageName] = measurement.Str(imageTag)
			}
		}
	}

	slog.Debug("collected container images", slog.Int("count", len(images)))
	return images, nil
}

// parseImageReference splits an image reference into name and tag.
// Example: "nvcr.io/nvidia/cuda:12.6.3" -> ("nvcr.io/nvidia/cuda", "12.6.3")
func parseImageReference(image string) (name, tag string) {
	// Handle image@digest format
	if idx := strings.Index(image, "@"); idx != -1 {
		return image[:idx], image[idx+1:]
	}

	// Handle image:tag format
	parts := strings.Split(image, ":")
	if len(parts) == 2 {
		return parts[0], parts[1]
	}

	// Handle registry with port (e.g., localhost:5000/image:tag)
	if len(parts) > 2 {
		// Join all but last as name, last is tag
		return strings.Join(parts[:len(parts)-1], ":"), parts[len(parts)-1]
	}

	// No tag specified, default to "latest"
	return image, "latest"
}
