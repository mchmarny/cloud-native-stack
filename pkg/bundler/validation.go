package bundler

import (
	"context"
	"fmt"

	"github.com/NVIDIA/cloud-native-stack/pkg/measurement"
	"github.com/NVIDIA/cloud-native-stack/pkg/recipe"
)

// Validator defines the interface for validating recipes before bundling.
type Validator interface {
	// Validate checks if the recipe is valid for this bundler.
	Validate(ctx context.Context, recipe *recipe.Recipe) error
}

// ValidateRecipe validates a recipe against all registered bundlers that implement Validator.
func ValidateRecipe(ctx context.Context, recipe *recipe.Recipe) error {
	if recipe == nil {
		return fmt.Errorf("recipe cannot be nil")
	}

	if len(recipe.Measurements) == 0 {
		return fmt.Errorf("recipe has no measurements")
	}

	// Validate with all bundlers that implement Validator
	bundlers := defaultRegistry.GetAll()
	for bundlerType, b := range bundlers {
		if v, ok := b.(Validator); ok {
			if err := v.Validate(ctx, recipe); err != nil {
				return fmt.Errorf("validation failed for %s: %w", bundlerType, err)
			}
		}
	}

	return nil
}

// ValidateRecipeStructure performs basic structural validation.
func ValidateRecipeStructure(recipe *recipe.Recipe) error {
	if recipe == nil {
		return fmt.Errorf("recipe is nil")
	}

	if len(recipe.Measurements) == 0 {
		return fmt.Errorf("no measurements in recipe")
	}

	// Validate each measurement
	for i, m := range recipe.Measurements {
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
func ValidateMeasurementExists(recipe *recipe.Recipe, measurementType measurement.Type) error {
	for _, m := range recipe.Measurements {
		if m.Type == measurementType {
			return nil
		}
	}
	return fmt.Errorf("measurement type %s not found in recipe", measurementType)
}

// ValidateSubtypeExists checks if a specific subtype exists within a measurement.
func ValidateSubtypeExists(recipe *recipe.Recipe, measurementType measurement.Type, subtypeName string) error {
	for _, m := range recipe.Measurements {
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
