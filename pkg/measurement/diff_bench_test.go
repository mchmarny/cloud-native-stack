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

import "testing"

func BenchmarkCompare(b *testing.B) {
	m1 := Measurement{
		Type: TypeK8s,
		Subtypes: []Subtype{
			{
				Name: "cluster",
				Data: map[string]Reading{
					"version": Str("1.28.0"),
					"nodes":   Int(3),
					"ready":   Bool(true),
				},
			},
			{
				Name: "pod",
				Data: map[string]Reading{
					"count": Int(100),
					"ready": Int(95),
				},
			},
		},
	}

	m2 := Measurement{
		Type: TypeK8s,
		Subtypes: []Subtype{
			{
				Name: "cluster",
				Data: map[string]Reading{
					"version": Str("1.29.0"),
					"nodes":   Int(5),
					"ready":   Bool(true),
				},
			},
			{
				Name: "pod",
				Data: map[string]Reading{
					"count": Int(150),
					"ready": Int(140),
				},
			},
			{
				Name: "service",
				Data: map[string]Reading{
					"count": Int(50),
				},
			},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = Compare(m1, m2)
	}
}

func BenchmarkCompare_NoChanges(b *testing.B) {
	m := Measurement{
		Type: TypeK8s,
		Subtypes: []Subtype{
			{
				Name: "cluster",
				Data: map[string]Reading{
					"version": Str("1.28.0"),
					"nodes":   Int(3),
					"ready":   Bool(true),
				},
			},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = Compare(m, m)
	}
}

func BenchmarkCompare_ManySubtypes(b *testing.B) {
	// Create measurements with many subtypes
	subtypes1 := make([]Subtype, 50)
	subtypes2 := make([]Subtype, 50)

	for i := 0; i < 50; i++ {
		subtypes1[i] = Subtype{
			Name: "subtype" + string(rune(i)),
			Data: map[string]Reading{
				"value1": Int(i),
				"value2": Str("test"),
				"value3": Bool(i%2 == 0),
			},
		}
		subtypes2[i] = Subtype{
			Name: "subtype" + string(rune(i)),
			Data: map[string]Reading{
				"value1": Int(i + 1), // Changed
				"value2": Str("test"),
				"value3": Bool(i%2 == 0),
			},
		}
	}

	m1 := Measurement{Type: TypeK8s, Subtypes: subtypes1}
	m2 := Measurement{Type: TypeK8s, Subtypes: subtypes2}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = Compare(m1, m2)
	}
}
