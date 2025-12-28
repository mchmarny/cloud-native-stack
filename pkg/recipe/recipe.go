package recipe

import (
	"context"
	"fmt"

	"github.com/NVIDIA/cloud-native-stack/pkg/measurement"
	"github.com/NVIDIA/cloud-native-stack/pkg/recipe/header"
	"github.com/NVIDIA/cloud-native-stack/pkg/snapshotter"
)

// Validator defines the interface for validating recipes before bundling.
type Validator interface {
	// Validate checks if the recipe is valid for this bundler.
	Validate(ctx context.Context, recipe *Recipe) error
}

const (
	// RecipeAPIVersion is the current API version for recipes
	RecipeAPIVersion = "v1"
)

// Recommender defines the interface for generating recommendations based on snapshots and intent.
type Recommender interface {
	Recommend(ctx context.Context, intent IntentType, snap *snapshotter.Snapshot) (*Recipe, error)
}

// Recipe represents the recipe response structure.
type Recipe struct {
	header.Header `json:",inline" yaml:",inline"`

	Request      *Query                     `json:"request,omitempty" yaml:"request,omitempty"`
	MatchedRules []string                   `json:"matchedRules,omitempty" yaml:"matchedRules,omitempty"`
	Measurements []*measurement.Measurement `json:"measurements" yaml:"measurements"`
}

// Store holds base measurements for recipes.
type Store struct {
	Base     []*measurement.Measurement `json:"base" yaml:"base"`
	Overlays []*Overlay                 `json:"overlays" yaml:"overlays"`
}

// Overlay represents overlay measurements for specific scenarios.
type Overlay struct {
	Key   Query                      `json:"key" yaml:"key"`
	Types []*measurement.Measurement `json:"types" yaml:"types"`
}

// Validate validates a recipe against all registered bundlers that implement Validator.
func (v *Recipe) Validate() error {
	if v == nil {
		return fmt.Errorf("recipe cannot be nil")
	}

	if len(v.Measurements) == 0 {
		return fmt.Errorf("recipe has no measurements")
	}

	return nil
}

// ValidateStructure performs basic structural validation.
func (v *Recipe) ValidateStructure() error {
	if err := v.Validate(); err != nil {
		return err
	}

	// Validate each measurement
	for i, m := range v.Measurements {
		if m == nil {
			return fmt.Errorf("measurement at index %d is nil", i)
		}

		if m.Type == "" {
			return fmt.Errorf("measurement at index %d has empty type", i)
		}

		if len(m.Subtypes) == 0 {
			return fmt.Errorf("measurement type %s has no subtypes", m.Type)
		}

		// Validate subtypes
		for j, st := range m.Subtypes {
			if st.Name == "" {
				return fmt.Errorf("subtype at index %d in measurement %s has empty name", j, m.Type)
			}

			if st.Data == nil {
				return fmt.Errorf("subtype %s in measurement %s has nil data", st.Name, m.Type)
			}
		}
	}

	return nil
}

// ValidateMeasurementExists checks if a specific measurement type exists.
func (v *Recipe) ValidateMeasurementExists(measurementType measurement.Type) error {
	if err := v.ValidateStructure(); err != nil {
		return err
	}

	for _, m := range v.Measurements {
		if m.Type == measurementType {
			return nil
		}
	}
	return fmt.Errorf("measurement type %s not found in recipe", measurementType)
}

// ValidateSubtypeExists checks if a specific subtype exists within a measurement.
func (v *Recipe) ValidateSubtypeExists(measurementType measurement.Type, subtypeName string) error {
	if err := v.ValidateMeasurementExists(measurementType); err != nil {
		return err
	}

	for _, m := range v.Measurements {
		if m.Type == measurementType {
			for _, st := range m.Subtypes {
				if st.Name == subtypeName {
					return nil
				}
			}
			return fmt.Errorf("subtype %s not found in measurement type %s", subtypeName, measurementType)
		}
	}
	return fmt.Errorf("measurement type %s not found in recipe", measurementType)
}

// ValidateRequiredKeys checks if required keys exist in a subtype's data.
func ValidateRequiredKeys(subtype *measurement.Subtype, requiredKeys []string) error {
	if subtype == nil {
		return fmt.Errorf("subtype is nil")
	}

	for _, key := range requiredKeys {
		if _, exists := subtype.Data[key]; !exists {
			return fmt.Errorf("required key %s not found in subtype %s", key, subtype.Name)
		}
	}

	return nil
}
