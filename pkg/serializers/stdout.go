package serializers

import (
	"encoding/json"
	"fmt"
)

// StdoutSerializer is a serializer that outputs snapshot data to stdout in JSON format.
type StdoutSerializer struct {
}

// Serialize outputs the given snapshot data to stdout in JSON format.
// It implements the Serializer interface.
func (s *StdoutSerializer) Serialize(snapshot any) error {
	j, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to serialize to json: %w", err)
	}

	fmt.Println(string(j))
	return nil
}
