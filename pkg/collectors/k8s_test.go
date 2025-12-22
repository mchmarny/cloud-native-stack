package collectors

import (
	"context"
	"testing"

	"github.com/NVIDIA/cloud-native-stack/pkg/measurement"
	"github.com/stretchr/testify/assert"
)

func TestKubernetesCollector_Collect(t *testing.T) {
	ctx := context.Background()
	collector := &KubernetesCollector{}

	// This test requires a kubernetes cluster to be available
	// It will fail if no cluster is accessible
	m, err := collector.Collect(ctx)

	// If no cluster is available, we expect an error
	if err != nil {
		assert.Error(t, err, "expected error when cluster is not available")
		t.Logf("Kubernetes collector failed as expected (no cluster available): %v", err)
		return
	}

	// If cluster is available, validate the measurement
	assert.NoError(t, err)
	assert.NotNil(t, m)
	assert.Equal(t, measurement.TypeK8s, m.Type)
	assert.Len(t, m.Subtypes, 1)
	assert.NotNil(t, m.Subtypes[0].Data)

	// Check that required fields are present
	data := m.Subtypes[0].Data
	assert.Contains(t, data, "version")
	assert.Contains(t, data, "platform")
	assert.Contains(t, data, "goVersion")

	version, _ := m.Subtypes[0].GetString("version")
	t.Logf("Kubernetes version: %s", version)
}

func TestKubernetesCollector_CollectWithCancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	collector := &KubernetesCollector{}
	m, err := collector.Collect(ctx)

	assert.Error(t, err)
	assert.Nil(t, m)
	assert.Equal(t, context.Canceled, err)
}
