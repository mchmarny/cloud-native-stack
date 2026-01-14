package gpu

import (
	"context"
	"encoding/xml"
	"os"
	"os/exec"
	"testing"

	"github.com/NVIDIA/cloud-native-stack/pkg/measurement"
)

func TestParseNvidiaSMILog(t *testing.T) {
	data, err := os.ReadFile("gpu.xml")
	if err != nil {
		t.Skipf("smi.xml not available: %v", err)
	}

	var d NVSMIDevice
	if err := xml.Unmarshal(data, &d); err != nil {
		t.Fatalf("failed to unmarshal XML: %v", err)
	}

	// Basic validations
	if d.Timestamp == "" {
		t.Error("expected timestamp to be set")
	}
	if d.DriverVersion == "" {
		t.Error("expected driverVersion to be set")
	}
	if d.CudaVersion == "" {
		t.Error("expected cudaVersion to be set")
	}
	if len(d.GPUs) != 8 {
		t.Error("expected 8 GPUs to be present")
	}
	for _, gpu := range d.GPUs {
		if gpu.Serial == "" {
			t.Error("expected GPU serial to be set")
		}

		if gpu.ProductName == "" {
			t.Error("expected GPU productName to be set")
		}
		if gpu.UUID == "" {
			t.Error("expected GPU UUID to be set")
		}
		if gpu.FbMemoryUsage.Total == "" {
			t.Error("expected fbMemoryUsage.total to be set")
		}
	}
}

func TestNoGPUMeasurement(t *testing.T) {
	m := noGPUMeasurement()

	if m == nil {
		t.Fatal("expected non-nil measurement")
	}

	if m.Type != measurement.TypeGPU {
		t.Errorf("expected type %q, got %q", measurement.TypeGPU, m.Type)
	}

	if len(m.Subtypes) != 1 {
		t.Fatalf("expected 1 subtype, got %d", len(m.Subtypes))
	}

	if m.Subtypes[0].Name != "smi" {
		t.Errorf("expected subtype name %q, got %q", "smi", m.Subtypes[0].Name)
	}

	gpuCount, ok := m.Subtypes[0].Data[measurement.KeyGPUCount]
	if !ok {
		t.Fatal("expected gpu-count key in data")
	}

	gpuCountVal, ok := gpuCount.Any().(int)
	if !ok {
		t.Fatalf("expected gpu-count to be int, got %T", gpuCount.Any())
	}

	if gpuCountVal != 0 {
		t.Errorf("expected gpu-count=0, got %d", gpuCountVal)
	}
}

func TestCollector_GracefulDegradation_WhenNvidiaSmiMissing(t *testing.T) {
	// Skip if nvidia-smi is actually available (we can't test graceful degradation)
	if _, err := exec.LookPath(nvidiaSMICommand); err == nil {
		t.Skip("nvidia-smi is available, skipping graceful degradation test")
	}

	collector := &Collector{}
	ctx := context.Background()

	m, err := collector.Collect(ctx)

	// Should NOT return an error
	if err != nil {
		t.Fatalf("expected no error when nvidia-smi missing, got: %v", err)
	}

	// Should return a valid measurement
	if m == nil {
		t.Fatal("expected non-nil measurement")
	}

	if m.Type != measurement.TypeGPU {
		t.Errorf("expected type %q, got %q", measurement.TypeGPU, m.Type)
	}

	// Should indicate 0 GPUs
	if len(m.Subtypes) < 1 {
		t.Fatal("expected at least 1 subtype")
	}

	gpuCount, ok := m.Subtypes[0].Data[measurement.KeyGPUCount]
	if !ok {
		t.Fatal("expected gpu-count key in data")
	}

	gpuCountVal, ok := gpuCount.Any().(int)
	if !ok {
		t.Fatalf("expected gpu-count to be int, got %T", gpuCount.Any())
	}

	if gpuCountVal != 0 {
		t.Errorf("expected gpu-count=0, got %d", gpuCountVal)
	}
}
