package recommender

import (
	"context"

	"github.com/NVIDIA/cloud-native-stack/pkg/measurement"
	"github.com/NVIDIA/cloud-native-stack/pkg/recipe"
	"github.com/NVIDIA/cloud-native-stack/pkg/snapshotter"
)

// Recommender defines the interface for generating recommendations based on snapshots and intent.
type Recommender interface {
	Recommend(ctx context.Context, intent recipe.IntentType, snap *snapshotter.Snapshot) (*Recommendation, error)
}

// Recommendation is the structure representing recommended configuration for a given set of
// environment configurations and intent. The environment configurations are derived from the
// provided snapshot.
type Recommendation struct {
	// Kind is the type of the recommendation object.
	Kind string `json:"kind,omitempty" yaml:"kind,omitempty"`

	// APIVersion is the API version of the snapshot object.
	APIVersion string `json:"apiVersion,omitempty" yaml:"apiVersion,omitempty"`

	// Metadata contains key-value pairs with metadata about the snapshot.
	Metadata map[string]string `json:"metadata,omitempty" yaml:"metadata,omitempty"`

	// Measurements contains the collected measurements from various collectors.
	Measurements []*measurement.Measurement `json:"measurements" yaml:"measurements"`
}
