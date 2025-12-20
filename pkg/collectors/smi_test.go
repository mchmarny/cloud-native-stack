package collectors

import (
	"encoding/xml"
	"os"
	"testing"
)

func TestParseNvidiaSMILog(t *testing.T) {
	data, err := os.ReadFile("smi.xml")
	if err != nil {
		t.Fatalf("failed to read XML file: %v", err)
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
