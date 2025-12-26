package collector

import (
	"context"
	"testing"

	"github.com/NVIDIA/cloud-native-stack/pkg/collector/systemd"
)

func TestDefaultCollectorFactory_CreateSystemDCollector(t *testing.T) {
	factory := NewDefaultFactory()
	factory.SystemDServices = []string{"test.service"}

	col := factory.CreateSystemDCollector()
	if col == nil {
		t.Fatal("Expected non-nil collector")
	}

	// Verify it's configured correctly
	systemdCollector, ok := col.(*systemd.Collector)
	if !ok {
		t.Fatal("Expected *systemd.SystemDCollector")
	}

	if len(systemdCollector.Services) != 1 || systemdCollector.Services[0] != "test.service" {
		t.Errorf("Expected [test.service], got %v", systemdCollector.Services)
	}
}

func TestDefaultCollectorFactory_CreateOSCollector(t *testing.T) {
	factory := NewDefaultFactory()

	collector := factory.CreateOSCollector()
	if collector == nil {
		t.Fatal("Expected non-nil collector")
	}

	ctx := context.TODO()
	_, err := collector.Collect(ctx)
	if err != nil {
		t.Logf("Collect returned error (acceptable): %v", err)
	}
}

func TestDefaultCollectorFactory_AllCollectors(t *testing.T) {
	factory := NewDefaultFactory()

	collectorFuncs := []func() Collector{
		factory.CreateSystemDCollector,
		factory.CreateOSCollector,
	}

	for i, createFunc := range collectorFuncs {
		collector := createFunc()
		if collector == nil {
			t.Errorf("Collector %d returned nil", i)
		}
	}
}
