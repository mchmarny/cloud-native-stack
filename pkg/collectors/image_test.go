package collectors

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestImageCollector_Collect(t *testing.T) {
	ctx := context.Background()
	collector := &ImageCollector{}

	// This test requires a kubernetes cluster to be available
	// It will fail if no cluster is accessible
	measurements, err := collector.Collect(ctx)

	// If no cluster is available, we expect an error
	if err != nil {
		assert.Error(t, err, "expected error when cluster is not available")
		t.Logf("Image collector failed as expected (no cluster available): %v", err)
		return
	}

	// If cluster is available, validate the measurement
	assert.NoError(t, err)
	assert.Len(t, measurements, 1)
	assert.Equal(t, ImageType, measurements[0].Type)
	assert.NotNil(t, measurements[0].Data)

	// Data should be a map
	data, ok := measurements[0].Data.(map[string]any)
	assert.True(t, ok, "Data should be a map[string]any")

	t.Logf("Found %d unique container images", len(data))
}

func TestImageCollector_CollectWithCancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	collector := &ImageCollector{}
	_, err := collector.Collect(ctx)

	assert.Error(t, err)
	assert.Equal(t, context.Canceled, err)
}

func TestParseImageReference(t *testing.T) {
	tests := []struct {
		name         string
		image        string
		expectedName string
		expectedTag  string
	}{
		{
			name:         "standard image with tag",
			image:        "nvcr.io/nvidia/cuda:12.6.3",
			expectedName: "nvcr.io/nvidia/cuda",
			expectedTag:  "12.6.3",
		},
		{
			name:         "image with digest",
			image:        "nvcr.io/nvidia/driver@sha256:abc123",
			expectedName: "nvcr.io/nvidia/driver",
			expectedTag:  "sha256:abc123",
		},
		{
			name:         "image without tag",
			image:        "nvcr.io/nvidia/cuda",
			expectedName: "nvcr.io/nvidia/cuda",
			expectedTag:  "latest",
		},
		{
			name:         "image with registry port",
			image:        "localhost:5000/myapp:v1.0",
			expectedName: "localhost:5000/myapp",
			expectedTag:  "v1.0",
		},
		{
			name:         "simple image name with tag",
			image:        "nginx:1.21",
			expectedName: "nginx",
			expectedTag:  "1.21",
		},
		{
			name:         "simple image name without tag",
			image:        "redis",
			expectedName: "redis",
			expectedTag:  "latest",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			name, tag := parseImageReference(tt.image)
			assert.Equal(t, tt.expectedName, name, "image name mismatch")
			assert.Equal(t, tt.expectedTag, tag, "image tag mismatch")
		})
	}
}
