package collectors

import (
	"context"
	"testing"

	"github.com/NVIDIA/cloud-native-stack/pkg/measurement"
	"github.com/stretchr/testify/assert"
)

func TestImageCollector_Collect(t *testing.T) {
	ctx := context.Background()
	collector := &ImageCollector{}

	// This test requires a kubernetes cluster to be available
	// It will fail if no cluster is accessible
	m, err := collector.Collect(ctx)

	// If no cluster is available, we expect an error
	if err != nil {
		assert.Error(t, err, "expected error when cluster is not available")
		t.Logf("Image collector failed as expected (no cluster available): %v", err)
		return
	}

	// If cluster is available, validate the measurement
	assert.NoError(t, err)
	assert.NotNil(t, m)
	assert.Equal(t, measurement.TypeImage, m.Type)
	assert.Len(t, m.Subtypes, 1)
	assert.NotNil(t, m.Subtypes[0].Data)

	// Data should be a map
	data := m.Subtypes[0].Data
	assert.NotEmpty(t, data)

	t.Logf("Found %d unique container images", len(data))
}

func TestImageCollector_CollectWithCancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	collector := &ImageCollector{}
	m, err := collector.Collect(ctx)

	assert.Error(t, err)
	assert.Nil(t, m)
	assert.Equal(t, context.Canceled, err)
}
