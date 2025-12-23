package recommendation

import (
	"testing"

	"github.com/NVIDIA/cloud-native-stack/pkg/measurement"
	"gopkg.in/yaml.v3"
)

const (
	version1283    = "1.28.3"
	version1290    = "1.29.0"
	version550     = "550"
	measurementKey = "version"
)

func TestBuildRecommendationMergesMatchingOverlay(t *testing.T) {
	t.Cleanup(setRecommendationData(t, `base:
  - type: K8s
    subtypes:
      - subtype: control-plane
        data:
          version: "1.28.3"
overlays:
  - key:
      os: ubuntu
      service: eks
    types:
      - type: K8s
        subtypes:
          - subtype: control-plane
            data:
              version: "1.29.0"
          - subtype: worker
            data:
              version: "1.29.0"
      - type: GPU
        subtypes:
          - subtype: drivers
            data:
              version: "550"
  - key:
      os: cos
      service: gke
    types:
      - type: GPU
        subtypes:
          - subtype: drivers
            data:
              version: "999"
`))

	query := &Query{Os: OSUbuntu, Service: ServiceEKS}
	rec, err := buildRecommendation(query)
	if err != nil {
		t.Fatalf("buildRecommendation() error = %v", err)
	}

	if got := measurementValue(t, rec.Measurements, measurement.TypeK8s, "control-plane"); got != version1290 {
		t.Fatalf("expected control-plane version %s, got %v", version1290, got)
	}
	if got := measurementValue(t, rec.Measurements, measurement.TypeK8s, "worker"); got != version1290 {
		t.Fatalf("expected worker version %s, got %v", version1290, got)
	}
	if got := measurementValue(t, rec.Measurements, measurement.TypeGPU, "drivers"); got != version550 {
		t.Fatalf("expected GPU driver version %s, got %v", version550, got)
	}

	// Verify original base payload remains unchanged after merging.
	var store Store
	if err := yaml.Unmarshal(recommendationData, &store); err != nil {
		t.Fatalf("failed to unmarshal store: %v", err)
	}
	if got := measurementValue(t, store.Base, measurement.TypeK8s, "control-plane"); got != version1283 {
		t.Fatalf("expected base control-plane to remain %s, got %v", version1283, got)
	}
	if measurementValueOrNil(store.Base, measurement.TypeK8s, "worker") != nil {
		t.Fatalf("expected worker subtype to be absent in base payload")
	}
}

func TestBuildRecommendationIgnoresNonMatchingOverlay(t *testing.T) {
	t.Cleanup(setRecommendationData(t, `base:
  - type: K8s
    subtypes:
      - subtype: control-plane
        data:
          version: "1.28.3"
overlays:
  - key:
      service: gke
    types:
      - type: K8s
        subtypes:
          - subtype: control-plane
            data:
              version: "1.30.0"
`))

	query := &Query{Os: OSUbuntu, Service: ServiceEKS}
	rec, err := buildRecommendation(query)
	if err != nil {
		t.Fatalf("buildRecommendation() error = %v", err)
	}

	if got := measurementValue(t, rec.Measurements, measurement.TypeK8s, "control-plane"); got != version1283 {
		t.Fatalf("expected control-plane version %s, got %v", version1283, got)
	}
	if measurementValueOrNil(rec.Measurements, measurement.TypeGPU, "drivers") != nil {
		t.Fatalf("did not expect GPU measurement to be added")
	}
}

func TestCloneMeasurementsDeepCopiesData(t *testing.T) {
	original := []*measurement.Measurement{
		{
			Type: measurement.TypeK8s,
			Subtypes: []measurement.Subtype{
				{
					Name: "control-plane",
					Data: map[string]measurement.Reading{"version": measurement.Str("1.28.3")},
				},
			},
		},
	}

	cloned := cloneMeasurements(original)
	if len(cloned) != len(original) {
		t.Fatalf("expected %d cloned measurements, got %d", len(original), len(cloned))
	}
	cloned[0].Subtypes[0].Data[measurementKey] = measurement.Str("mutated")
	if got := original[0].Subtypes[0].Data[measurementKey].Any(); got != version1283 {
		t.Fatalf("expected original data to remain %s, got %v", version1283, got)
	}
}

func TestMergeOverlayMeasurementsHandlesExistingAndNewTypes(t *testing.T) {
	base := []*measurement.Measurement{
		{
			Type: measurement.TypeK8s,
			Subtypes: []measurement.Subtype{
				{
					Name: "control-plane",
					Data: map[string]measurement.Reading{"version": measurement.Str("1.28.3")},
				},
			},
		},
	}
	baseClone := cloneMeasurements(base)
	index := indexMeasurementsByType(baseClone)

	overlays := []*measurement.Measurement{
		nil,
		{
			Type: measurement.TypeK8s,
			Subtypes: []measurement.Subtype{
				{
					Name: "control-plane",
					Data: map[string]measurement.Reading{"version": measurement.Str("1.29.0")},
				},
				{
					Name: "worker",
					Data: map[string]measurement.Reading{"version": measurement.Str("1.29.0")},
				},
				{Name: "ignored", Data: nil},
			},
		},
		{
			Type: measurement.TypeGPU,
			Subtypes: []measurement.Subtype{
				{
					Name: "drivers",
					Data: map[string]measurement.Reading{"version": measurement.Str("550")},
				},
			},
		},
	}

	merged, idx := mergeOverlayMeasurements(baseClone, index, overlays)
	if len(merged) != 2 {
		t.Fatalf("expected 2 merged measurements, got %d", len(merged))
	}
	if _, ok := idx[measurement.TypeGPU]; !ok {
		t.Fatalf("expected GPU measurement to be indexed")
	}
	if got := measurementValue(t, merged, measurement.TypeK8s, "control-plane"); got != version1290 {
		t.Fatalf("expected control-plane override to %s, got %v", version1290, got)
	}
	if got := measurementValue(t, merged, measurement.TypeK8s, "worker"); got != version1290 {
		t.Fatalf("expected worker subtype to be added, got %v", got)
	}
}

func setRecommendationData(t *testing.T, payload string) func() {
	t.Helper()
	original := recommendationData
	recommendationData = []byte(payload)
	return func() {
		recommendationData = original
	}
}

func measurementValue(t *testing.T, measurements []*measurement.Measurement, typ measurement.Type, subtype string) interface{} {
	t.Helper()
	for _, m := range measurements {
		if m == nil || m.Type != typ {
			continue
		}
		for _, st := range m.Subtypes {
			if st.Name != subtype {
				continue
			}
			if st.Data == nil {
				t.Fatalf("subtype %s has nil data", subtype)
			}
			if reading, ok := st.Data[measurementKey]; ok {
				return reading.Any()
			}
		}
	}
	t.Fatalf("value %s/%s/%s not found", typ, subtype, measurementKey)
	return nil
}

func measurementValueOrNil(measurements []*measurement.Measurement, typ measurement.Type, subtype string) interface{} {
	for _, m := range measurements {
		if m == nil || m.Type != typ {
			continue
		}
		for _, st := range m.Subtypes {
			if st.Name != subtype {
				continue
			}
			if st.Data == nil {
				return nil
			}
			if reading, ok := st.Data[measurementKey]; ok {
				return reading.Any()
			}
		}
	}
	return nil
}
