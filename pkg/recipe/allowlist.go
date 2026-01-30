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
	"log/slog"
	"os"
	"slices"
	"strings"

	cnserrors "github.com/NVIDIA/cloud-native-stack/pkg/errors"
)

// Environment variable names for allowlist configuration.
const (
	EnvAllowedAccelerators = "CNS_ALLOWED_ACCELERATORS"
	EnvAllowedServices     = "CNS_ALLOWED_SERVICES"
	EnvAllowedIntents      = "CNS_ALLOWED_INTENTS"
	EnvAllowedOSTypes      = "CNS_ALLOWED_OS"
)

// AllowLists defines which criteria values are permitted for API requests.
// An empty or nil slice means all values are allowed for that criteria type.
// This is used by the API server to restrict which values can be requested,
// while the CLI always allows all values.
type AllowLists struct {
	// Accelerators is the list of allowed accelerator types (e.g., "h100", "l40").
	// If empty, all accelerator types are allowed.
	Accelerators []CriteriaAcceleratorType

	// Services is the list of allowed service types (e.g., "eks", "gke").
	// If empty, all service types are allowed.
	Services []CriteriaServiceType

	// Intents is the list of allowed intent types (e.g., "training", "inference").
	// If empty, all intent types are allowed.
	Intents []CriteriaIntentType

	// OSTypes is the list of allowed OS types (e.g., "ubuntu", "rhel").
	// If empty, all OS types are allowed.
	OSTypes []CriteriaOSType
}

// IsEmpty returns true if no allowlists are configured (all values allowed).
func (a *AllowLists) IsEmpty() bool {
	if a == nil {
		return true
	}
	return len(a.Accelerators) == 0 &&
		len(a.Services) == 0 &&
		len(a.Intents) == 0 &&
		len(a.OSTypes) == 0
}

// AcceleratorStrings returns the allowed accelerator types as strings.
func (a *AllowLists) AcceleratorStrings() []string {
	if a == nil {
		return nil
	}
	return acceleratorTypesToStrings(a.Accelerators)
}

// ServiceStrings returns the allowed service types as strings.
func (a *AllowLists) ServiceStrings() []string {
	if a == nil {
		return nil
	}
	return serviceTypesToStrings(a.Services)
}

// IntentStrings returns the allowed intent types as strings.
func (a *AllowLists) IntentStrings() []string {
	if a == nil {
		return nil
	}
	return intentTypesToStrings(a.Intents)
}

// OSTypeStrings returns the allowed OS types as strings.
func (a *AllowLists) OSTypeStrings() []string {
	if a == nil {
		return nil
	}
	return osTypesToStrings(a.OSTypes)
}

// ValidateCriteria checks if the given criteria values are permitted by the allowlists.
// Returns nil if validation passes, or an error with details about what value is not allowed.
// The "any" value is always allowed regardless of the allowlist configuration.
func (a *AllowLists) ValidateCriteria(c *Criteria) error {
	if a == nil || c == nil {
		return nil
	}

	slog.Debug("evaluating criteria against allowlists",
		"criteria_accelerator", string(c.Accelerator),
		"criteria_service", string(c.Service),
		"criteria_intent", string(c.Intent),
		"criteria_os", string(c.OS),
		"allowed_accelerators", a.AcceleratorStrings(),
		"allowed_services", a.ServiceStrings(),
		"allowed_intents", a.IntentStrings(),
		"allowed_os_types", a.OSTypeStrings(),
	)

	// Check accelerator
	if len(a.Accelerators) > 0 && c.Accelerator != CriteriaAcceleratorAny {
		if !slices.Contains(a.Accelerators, c.Accelerator) {
			return cnserrors.WrapWithContext(
				cnserrors.ErrCodeInvalidRequest,
				"accelerator type not allowed",
				nil,
				map[string]any{
					"requested": string(c.Accelerator),
					"allowed":   acceleratorTypesToStrings(a.Accelerators),
				},
			)
		}
	}

	// Check service
	if len(a.Services) > 0 && c.Service != CriteriaServiceAny {
		if !slices.Contains(a.Services, c.Service) {
			return cnserrors.WrapWithContext(
				cnserrors.ErrCodeInvalidRequest,
				"service type not allowed",
				nil,
				map[string]any{
					"requested": string(c.Service),
					"allowed":   serviceTypesToStrings(a.Services),
				},
			)
		}
	}

	// Check intent
	if len(a.Intents) > 0 && c.Intent != CriteriaIntentAny {
		if !slices.Contains(a.Intents, c.Intent) {
			return cnserrors.WrapWithContext(
				cnserrors.ErrCodeInvalidRequest,
				"intent type not allowed",
				nil,
				map[string]any{
					"requested": string(c.Intent),
					"allowed":   intentTypesToStrings(a.Intents),
				},
			)
		}
	}

	// Check OS
	if len(a.OSTypes) > 0 && c.OS != CriteriaOSAny {
		if !slices.Contains(a.OSTypes, c.OS) {
			return cnserrors.WrapWithContext(
				cnserrors.ErrCodeInvalidRequest,
				"OS type not allowed",
				nil,
				map[string]any{
					"requested": string(c.OS),
					"allowed":   osTypesToStrings(a.OSTypes),
				},
			)
		}
	}

	return nil
}

// ParseAllowListsFromEnv parses allowlist configuration from environment variables.
// Returns nil if no allowlist environment variables are set.
// Environment variables:
//   - CNS_ALLOWED_ACCELERATORS: comma-separated list of accelerator types (e.g., "h100,l40")
//   - CNS_ALLOWED_SERVICES: comma-separated list of service types (e.g., "eks,gke")
//   - CNS_ALLOWED_INTENTS: comma-separated list of intent types (e.g., "training,inference")
//   - CNS_ALLOWED_OS: comma-separated list of OS types (e.g., "ubuntu,rhel")
//
// Invalid values in the environment variables are skipped with a warning logged.
func ParseAllowListsFromEnv() (*AllowLists, error) {
	al := &AllowLists{}

	// Parse accelerators
	if v := os.Getenv(EnvAllowedAccelerators); v != "" {
		accelerators, err := parseAcceleratorList(v)
		if err != nil {
			return nil, cnserrors.WrapWithContext(
				cnserrors.ErrCodeInvalidRequest,
				"invalid "+EnvAllowedAccelerators,
				err,
				map[string]any{"value": v},
			)
		}
		al.Accelerators = accelerators
	}

	// Parse services
	if v := os.Getenv(EnvAllowedServices); v != "" {
		services, err := parseServiceList(v)
		if err != nil {
			return nil, cnserrors.WrapWithContext(
				cnserrors.ErrCodeInvalidRequest,
				"invalid "+EnvAllowedServices,
				err,
				map[string]any{"value": v},
			)
		}
		al.Services = services
	}

	// Parse intents
	if v := os.Getenv(EnvAllowedIntents); v != "" {
		intents, err := parseIntentList(v)
		if err != nil {
			return nil, cnserrors.WrapWithContext(
				cnserrors.ErrCodeInvalidRequest,
				"invalid "+EnvAllowedIntents,
				err,
				map[string]any{"value": v},
			)
		}
		al.Intents = intents
	}

	// Parse OS types
	if v := os.Getenv(EnvAllowedOSTypes); v != "" {
		osTypes, err := parseOSList(v)
		if err != nil {
			return nil, cnserrors.WrapWithContext(
				cnserrors.ErrCodeInvalidRequest,
				"invalid "+EnvAllowedOSTypes,
				err,
				map[string]any{"value": v},
			)
		}
		al.OSTypes = osTypes
	}

	// Return nil if no allowlists configured (empty struct means all allowed)
	if al.IsEmpty() {
		return nil, nil //nolint:nilnil // nil allowlist means all values allowed, not an error
	}

	return al, nil
}

// parseAcceleratorList parses a comma-separated list of accelerator types.
func parseAcceleratorList(s string) ([]CriteriaAcceleratorType, error) {
	var result []CriteriaAcceleratorType
	for _, v := range strings.Split(s, ",") {
		v = strings.TrimSpace(v)
		if v == "" {
			continue
		}
		at, err := ParseCriteriaAcceleratorType(v)
		if err != nil {
			return nil, err
		}
		// Skip "any" in allowlist - it doesn't make sense to allow only "any"
		if at != CriteriaAcceleratorAny {
			result = append(result, at)
		}
	}
	return result, nil
}

// parseServiceList parses a comma-separated list of service types.
func parseServiceList(s string) ([]CriteriaServiceType, error) {
	var result []CriteriaServiceType
	for _, v := range strings.Split(s, ",") {
		v = strings.TrimSpace(v)
		if v == "" {
			continue
		}
		st, err := ParseCriteriaServiceType(v)
		if err != nil {
			return nil, err
		}
		if st != CriteriaServiceAny {
			result = append(result, st)
		}
	}
	return result, nil
}

// parseIntentList parses a comma-separated list of intent types.
func parseIntentList(s string) ([]CriteriaIntentType, error) {
	var result []CriteriaIntentType
	for _, v := range strings.Split(s, ",") {
		v = strings.TrimSpace(v)
		if v == "" {
			continue
		}
		it, err := ParseCriteriaIntentType(v)
		if err != nil {
			return nil, err
		}
		if it != CriteriaIntentAny {
			result = append(result, it)
		}
	}
	return result, nil
}

// parseOSList parses a comma-separated list of OS types.
func parseOSList(s string) ([]CriteriaOSType, error) {
	var result []CriteriaOSType
	for _, v := range strings.Split(s, ",") {
		v = strings.TrimSpace(v)
		if v == "" {
			continue
		}
		ot, err := ParseCriteriaOSType(v)
		if err != nil {
			return nil, err
		}
		if ot != CriteriaOSAny {
			result = append(result, ot)
		}
	}
	return result, nil
}

// Helper functions to convert typed slices to string slices for error messages.

func acceleratorTypesToStrings(types []CriteriaAcceleratorType) []string {
	result := make([]string, len(types))
	for i, t := range types {
		result[i] = string(t)
	}
	return result
}

func serviceTypesToStrings(types []CriteriaServiceType) []string {
	result := make([]string, len(types))
	for i, t := range types {
		result[i] = string(t)
	}
	return result
}

func intentTypesToStrings(types []CriteriaIntentType) []string {
	result := make([]string, len(types))
	for i, t := range types {
		result[i] = string(t)
	}
	return result
}

func osTypesToStrings(types []CriteriaOSType) []string {
	result := make([]string, len(types))
	for i, t := range types {
		result[i] = string(t)
	}
	return result
}
