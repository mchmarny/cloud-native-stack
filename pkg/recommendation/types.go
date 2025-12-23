package recommendation

import (
	"time"

	"github.com/NVIDIA/cloud-native-stack/pkg/measurement"
)

const (
	// RecommendationAPIVersion is the current API version for recommendations
	RecommendationAPIVersion = "v1"
)

// Recommendation represents the recommendation response structure.
type Recommendation struct {
	Request        *Query                     `json:"request"`
	MatchedRuleID  string                     `json:"matchedRuleId"`
	PayloadVersion string                     `json:"payloadVersion"`
	GeneratedAt    time.Time                  `json:"generatedAt"`
	Measurements   []*measurement.Measurement `json:"measurements"`
}

// RecommendationStore holds base measurements for recommendations.
type Store struct {
	Base     []*measurement.Measurement `json:"base" yaml:"base"`
	Overlays []*Overlay                 `json:"overlays" yaml:"overlays"`
}

// RecommendationOverlay represents overlay measurements for specific scenarios.
type Overlay struct {
	Key   Query                      `json:"key" yaml:"key"`
	Types []*measurement.Measurement `json:"types" yaml:"types"`
}
