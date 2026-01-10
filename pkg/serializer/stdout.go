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
func (s *StdoutSerializer) Serialize(ctx context.Context, snapshot any) error {
	j, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to serialize to json: %w", err)
	}

	fmt.Println(string(j))
	return nil
}
