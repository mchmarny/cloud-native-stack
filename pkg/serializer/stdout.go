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

package serializer

import (
	"context"
	"encoding/json"
	"fmt"
)

// StdoutSerializer is a serializer that outputs snapshot data to stdout in JSON format.
//
// Deprecated: Use Writer with NewStdoutWriter(FormatJSON) instead for more flexibility
// and consistent API. StdoutSerializer is maintained for backward compatibility.
//
// Example migration:
//
//	// Old:
//	// s := &StdoutSerializer{}
//	// s.Serialize(data)
//
//	// New:
//	// w := NewStdoutWriter(FormatJSON)
//	// w.Serialize(data)
type StdoutSerializer struct {
}

// Serialize outputs the given snapshot data to stdout in JSON format.
// It implements the Serializer interface.
// Context is provided for consistency but not actively used for stdout writes.
func (s *StdoutSerializer) Serialize(_ context.Context, snapshot any) error {
	j, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to serialize to json: %w", err)
	}

	fmt.Println(string(j))
	return nil
}
