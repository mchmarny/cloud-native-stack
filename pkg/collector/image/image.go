package image

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/NVIDIA/cloud-native-stack/pkg/k8s/client"
	"github.com/NVIDIA/cloud-native-stack/pkg/measurement"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// Collector collects information about container images running in the cluster.
type Collector struct {
	Clientset kubernetes.Interface
}

// Collect retrieves unique container images from all pods in the cluster.
// This provides a snapshot of deployed images for cluster comparison.
func (i *Collector) Collect(ctx context.Context) (*measurement.Measurement, error) {
	// Check if context is canceled
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	k8sClient, err := i.getClient()
	if err != nil {
		return nil, err
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

func (i *Collector) getClient() (kubernetes.Interface, error) {
	if i.Clientset != nil {
		return i.Clientset, nil
	}
	k8sClient, _, err := client.GetKubeClient("")
	if err != nil {
		return nil, fmt.Errorf("failed to get kubernetes client: %w", err)
	}
	return k8sClient, nil
}

// collectContainerImages extracts unique container images from all pods.
func (i *Collector) collectContainerImages(ctx context.Context, k8sClient kubernetes.Interface) (map[string]measurement.Reading, error) {
	pods, err := k8sClient.CoreV1().Pods("").List(ctx, v1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list pods: %w", err)
	}

	images := make(map[string]measurement.Reading)
	recordImage := func(imageRef, location string) {
		if imageRef == "" {
			return
		}
		if _, exists := images[imageRef]; exists {
			return
		}
		images[imageRef] = measurement.Str(location)
	}
	for _, pod := range pods.Items {
		// Check for context cancellation
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		locationPrefix := fmt.Sprintf("%s/%s", pod.Namespace, pod.Name)

		for _, container := range pod.Spec.Containers {
			recordImage(container.Image, fmt.Sprintf("%s:%s", locationPrefix, container.Name))
		}
		for _, container := range pod.Spec.InitContainers {
			recordImage(container.Image, fmt.Sprintf("%s:init-%s", locationPrefix, container.Name))
		}
		for _, container := range pod.Spec.EphemeralContainers {
			recordImage(container.Image, fmt.Sprintf("%s:ephemeral-%s", locationPrefix, container.Name))
		}
	}

	slog.Debug("collected container images", slog.Int("count", len(images)))
	return images, nil
}
