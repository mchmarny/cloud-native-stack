package recommendation

import (
	_ "embed"
	"fmt"
	"sync"
	"time"

	"github.com/NVIDIA/cloud-native-stack/pkg/measurement"
	"gopkg.in/yaml.v3"
)

var (
	//go:embed data/data-v1.yaml
	recommendationData []byte

	storeOnce   sync.Once
	cachedStore *Store
	storeErr    error

	defaultBuilder = &Builder{}
)

// Builder produces recommendation payloads and optionally exposes cache metadata
// (e.g., TTL) for higher layers like HTTP handlers.
type Builder struct {
	CacheTTL time.Duration
}

// BuildRecommendation creates a Recommendation based on the query using a shared
// default Builder instance. Prefer using Builder directly when custom settings
// like cache TTL are required.
func BuildRecommendation(q *Query) (*Recommendation, error) {
	return defaultBuilder.Build(q)
}

// Build creates a Recommendation payload for the provided query.
func (b *Builder) Build(q *Query) (*Recommendation, error) {
	if q == nil {
		return nil, fmt.Errorf("query cannot be nil")
	}

	store, err := loadStore()
	if err != nil {
		return nil, fmt.Errorf("failed to load recommendation store: %w", err)
	}

	r := &Recommendation{
		Request:        q,
		PayloadVersion: RecommendationAPIVersion,
		MatchedRules:   make([]string, 0),
		GeneratedAt:    time.Now().UTC(),
	}

	merged := cloneMeasurements(store.Base)
	index := indexMeasurementsByType(merged)

	for _, overlay := range store.Overlays {
		// overlays use Query as key, so matching queries inherit overlay-specific measurements
		if overlay.Key.IsMatch(q) {
			merged, index = mergeOverlayMeasurements(merged, index, overlay.Types)
			r.MatchedRules = append(r.MatchedRules, overlay.Key.String())
		}
	}

	r.Measurements = merged
	return r, nil
}

func loadStore() (*Store, error) {
	storeOnce.Do(func() {
		var store Store
		if err := yaml.Unmarshal(recommendationData, &store); err != nil {
			storeErr = fmt.Errorf("failed to unmarshal recommendation data: %w", err)
			return
		}
		cachedStore = &store
	})
	return cachedStore, storeErr
}

// cloneMeasurements creates deep copies of all measurements so we never mutate
// the shared store payload while tailoring responses.
func cloneMeasurements(list []*measurement.Measurement) []*measurement.Measurement {
	if len(list) == 0 {
		return nil
	}
	cloned := make([]*measurement.Measurement, 0, len(list))
	for _, m := range list {
		if m == nil {
			continue
		}
		cloned = append(cloned, cloneMeasurement(m))
	}
	return cloned
}

// cloneMeasurement duplicates a single measurement including all of its
// subtypes to protect original data from in-place updates.
func cloneMeasurement(m *measurement.Measurement) *measurement.Measurement {
	if m == nil {
		return nil
	}
	clone := &measurement.Measurement{
		Type:     m.Type,
		Subtypes: make([]measurement.Subtype, len(m.Subtypes)),
	}
	for i := range m.Subtypes {
		clone.Subtypes[i] = cloneSubtype(m.Subtypes[i])
	}
	return clone
}

// cloneSubtype duplicates an individual subtype and its key/value readings.
func cloneSubtype(st measurement.Subtype) measurement.Subtype {
	cloned := measurement.Subtype{
		Name: st.Name,
	}
	if len(st.Data) == 0 {
		return cloned
	}
	cloned.Data = make(map[string]measurement.Reading, len(st.Data))
	for k, v := range st.Data {
		cloned.Data[k] = v
	}
	return cloned
}

// indexMeasurementsByType builds an index for O(1) lookup when merging
// overlays by measurement type.
func indexMeasurementsByType(measurements []*measurement.Measurement) map[measurement.Type]*measurement.Measurement {
	index := make(map[measurement.Type]*measurement.Measurement, len(measurements))
	for _, m := range measurements {
		if m == nil {
			continue
		}
		index[m.Type] = m
	}
	return index
}

// mergeOverlayMeasurements folds overlay measurements into the base slice,
// appending new types and delegating to subtype merging when the type already exists.
func mergeOverlayMeasurements(base []*measurement.Measurement, index map[measurement.Type]*measurement.Measurement, overlays []*measurement.Measurement) ([]*measurement.Measurement, map[measurement.Type]*measurement.Measurement) {
	if len(overlays) == 0 {
		return base, index
	}
	for _, overlay := range overlays {
		if overlay == nil {
			continue
		}
		if target, ok := index[overlay.Type]; ok {
			mergeMeasurementSubtypes(target, overlay)
			continue
		}
		cloned := cloneMeasurement(overlay)
		base = append(base, cloned)
		index[cloned.Type] = cloned
	}
	return base, index
}

// mergeMeasurementSubtypes walks all subtypes so overlay data augments or
// overrides existing subtype readings.
func mergeMeasurementSubtypes(target, overlay *measurement.Measurement) {
	if target == nil || overlay == nil {
		return
	}
	subtypeIndex := make(map[string]*measurement.Subtype, len(target.Subtypes))
	for i := range target.Subtypes {
		st := &target.Subtypes[i]
		subtypeIndex[st.Name] = st
		if st.Data == nil {
			st.Data = make(map[string]measurement.Reading)
		}
	}
	for _, overlaySubtype := range overlay.Subtypes {
		if overlaySubtype.Data == nil {
			continue
		}
		if targetSubtype, ok := subtypeIndex[overlaySubtype.Name]; ok {
			if targetSubtype.Data == nil {
				targetSubtype.Data = make(map[string]measurement.Reading, len(overlaySubtype.Data))
			}
			for key, reading := range overlaySubtype.Data {
				targetSubtype.Data[key] = reading
			}
			continue
		}
		target.Subtypes = append(target.Subtypes, cloneSubtype(overlaySubtype))
		subtypeIndex[overlaySubtype.Name] = &target.Subtypes[len(target.Subtypes)-1]
	}
}
