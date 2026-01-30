// Copyright (c) 2025, NVIDIA CORPORATION.  All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package recipe

import (
	"context"
	"fmt"

	"github.com/NVIDIA/cloud-native-stack/pkg/header"
	"github.com/NVIDIA/cloud-native-stack/pkg/measurement"
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

// RequestInfo holds simplified request metadata for documentation purposes.
// This replaces the old Query type with just the fields needed for bundle documentation.
type RequestInfo struct {
	Os        string `json:"os,omitempty" yaml:"os,omitempty"`
	OsVersion string `json:"osVersion,omitempty" yaml:"osVersion,omitempty"`
	Service   string `json:"service,omitempty" yaml:"service,omitempty"`
	K8s       string `json:"k8s,omitempty" yaml:"k8s,omitempty"`
	GPU       string `json:"gpu,omitempty" yaml:"gpu,omitempty"`
	Intent    string `json:"intent,omitempty" yaml:"intent,omitempty"`
}

// Recipe represents the recipe response structure.
type Recipe struct {
	header.Header `json:",inline" yaml:",inline"`

	Request      *RequestInfo               `json:"request,omitempty" yaml:"request,omitempty"`
	MatchedRules []string                   `json:"matchedRules,omitempty" yaml:"matchedRules,omitempty"`
	Measurements []*measurement.Measurement `json:"measurements" yaml:"measurements"`
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
