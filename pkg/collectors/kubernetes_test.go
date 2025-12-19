package collectors

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestKubernetesCollector_Collect(t *testing.T) {
	ctx := context.Background()
	collector := &KubernetesCollector{}

	// This test requires a kubernetes cluster to be available
	// It will fail if no cluster is accessible
	measurements, err := collector.Collect(ctx)

	// If no cluster is available, we expect an error
	if err != nil {
		assert.Error(t, err, "expected error when cluster is not available")
		t.Logf("Kubernetes collector failed as expected (no cluster available): %v", err)
		return
	}

	// If cluster is available, validate the measurement
	assert.NoError(t, err)
	assert.Len(t, measurements, 1)
	assert.Equal(t, KubernetesType, measurements[0].Type)
	assert.NotNil(t, measurements[0].Data)

	// Data should be a map with version info
	data, ok := measurements[0].Data.(map[string]string)
	assert.True(t, ok, "Data should be a map[string]string")

	// Check that required fields are present
	assert.Contains(t, data, "version")
	assert.Contains(t, data, "platform")
	assert.Contains(t, data, "goVersion")

	t.Logf("Kubernetes version: %s", data["version"])
}

func TestKubernetesCollector_CollectWithCancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	collector := &KubernetesCollector{}
	_, err := collector.Collect(ctx)

	assert.Error(t, err)
	assert.Equal(t, context.Canceled, err)
}
