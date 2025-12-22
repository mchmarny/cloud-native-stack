package serializers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
)

// RespondJSON writes a JSON response with the given status code and data.
func RespondJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		fmt.Fprintf(os.Stderr, "failed to encode JSON response: %v\n", err)
	}
}
