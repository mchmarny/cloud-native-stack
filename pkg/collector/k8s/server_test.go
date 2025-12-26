package k8s

import (
	"context"
	"testing"

	"github.com/NVIDIA/cloud-native-stack/pkg/measurement"
	"github.com/stretchr/testify/assert"
)

func TestKubernetesCollector_Collect(t *testing.T) {
	t.Setenv("NODE_NAME", "test-node")

	ctx := context.TODO()
	collector := createTestCollector()

	m, err := collector.Collect(ctx)
	assert.NoError(t, err)
	assert.NotNil(t, m)
	assert.Equal(t, measurement.TypeK8s, m.Type)
	// Should have 4 subtypes: server, image, policy, and provider
	assert.Len(t, m.Subtypes, 4)

	// Find the server subtype
	var serverSubtype *measurement.Subtype
	for i := range m.Subtypes {
		if m.Subtypes[i].Name == "server" {
			serverSubtype = &m.Subtypes[i]
			break
		}
	}
	if !assert.NotNil(t, serverSubtype, "Expected to find server subtype") {
		return
	}

	data := serverSubtype.Data
	if assert.Len(t, data, 3) {
		if reading, ok := data["version"]; assert.True(t, ok) {
			assert.Equal(t, "v1.28.0", reading.Any())
		}
		if reading, ok := data["platform"]; assert.True(t, ok) {
			assert.Equal(t, "linux/amd64", reading.Any())
		}
		if reading, ok := data["goVersion"]; assert.True(t, ok) {
			assert.Equal(t, "go1.20.7", reading.Any())
		}
	}
}

func TestKubernetesCollector_CollectWithCancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.TODO())
	cancel() // Cancel immediately

	collector := createTestCollector()
	m, err := collector.Collect(ctx)

	assert.Error(t, err)
	assert.Nil(t, m)
	assert.Equal(t, context.Canceled, err)
}

// Helper function defined in image_test.go
// Reused here to avoid duplication across test files
