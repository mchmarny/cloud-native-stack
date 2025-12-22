package recommendation

import (
	"time"

	"github.com/NVIDIA/cloud-native-stack/pkg/measurement"
)

const (
	// RecommendationAPIVersion is the current API version for recommendations
	RecommendationAPIVersion = "2025-12-0"
)

type Recommendation struct {
	Request        *Query                     `json:"request"`
	MatchedRuleID  string                     `json:"matchedRuleId"`
	PayloadVersion string                     `json:"payloadVersion"`
	GeneratedAt    time.Time                  `json:"generatedAt"`
	Measurements   []*measurement.Measurement `json:"measurements"`
}
