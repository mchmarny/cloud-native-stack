package recommendation

import (
	// embed package is required to satisfy go:embed directives declared in this package.
	_ "embed"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/NVIDIA/cloud-native-stack/pkg/serializers"
)

// Builder handles recommendation requests and generates responses.
type Builder struct {
	CacheTTL time.Duration
}

func (b *Builder) Handle(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		slog.Error("method not allowed", "method", r.Method)
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	q, err := ParseQuery(r)
	if err != nil {
		slog.Error("failed to parse query", "error", err)
		http.Error(w, fmt.Sprintf("Bad Request: %v", err), http.StatusBadRequest)
		return
	}

	// Generate recommendation (stub implementation)
	resp, err := buildRecommendation(q)
	if err != nil {
		slog.Error("failed to build recommendation", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Set cache headers
	w.Header().Set("Cache-Control", fmt.Sprintf("public, max-age=%d", b.CacheTTL))

	serializers.RespondJSON(w, http.StatusOK, resp)
}
