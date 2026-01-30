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

package measurement

import (
	"fmt"
	"reflect"
)

// Compare compares two measurements and returns the differences in their subtypes.
// Returns subtypes from m2 that are new or have different data compared to m1.
// Each returned subtype contains only the keys that differ or are new.
func Compare(m1, m2 Measurement) ([]*Subtype, error) {
	if m1.Type != m2.Type {
		return nil, fmt.Errorf("cannot compare different measurement types: %q (%d subtypes) vs %q (%d subtypes)",
			m1.Type, len(m1.Subtypes), m2.Type, len(m2.Subtypes))
	}

	var diffs []*Subtype

	// Create a map for quick lookup of m1 subtypes
	m1Subtypes := make(map[string]*Subtype)
	for i := range m1.Subtypes {
		m1Subtypes[m1.Subtypes[i].Name] = &m1.Subtypes[i]
	}

	// Compare each subtype in m2
	for i := range m2.Subtypes {
		m2Sub := &m2.Subtypes[i]
		m1Sub, exists := m1Subtypes[m2Sub.Name]

		if !exists {
			// Subtype is new in m2, include all data
			diffs = append(diffs, m2Sub)
			continue
		}

		// Subtype exists in both, find differences in data
		diffData := make(map[string]Reading)
		for key, m2Value := range m2Sub.Data {
			m1Value, exists := m1Sub.Data[key]
			if !exists {
				// Key is new in m2
				diffData[key] = m2Value
				continue
			}

			// Compare values using reflect.DeepEqual to handle all scalar types
			if !reflect.DeepEqual(m1Value.Any(), m2Value.Any()) {
				diffData[key] = m2Value
			}
		}

		// If there are differences, add this subtype to diffs
		if len(diffData) > 0 {
			diffs = append(diffs, &Subtype{
				Name: m2Sub.Name,
				Data: diffData,
			})
		}
	}

	return diffs, nil
}
