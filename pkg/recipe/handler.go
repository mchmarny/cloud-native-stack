package recipe

import (
	"fmt"
	"log/slog"
	"net/http"

	"github.com/NVIDIA/cloud-native-stack/pkg/recipe/version"
	"github.com/NVIDIA/cloud-native-stack/pkg/serializer"
	"github.com/NVIDIA/cloud-native-stack/pkg/server"
)

var (
	recipeCacheTTLInSec = 600 // 10 minutes in seconds
)

// HandleRecipes processes recipe requests and returns recipes.
// It supports GET requests with query parameters to specify recipe criteria.
// The response is returned in JSON format with appropriate caching headers.
// Errors are handled and returned in a structured format.
func (b *Builder) HandleRecipes(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		server.WriteError(w, r, http.StatusMethodNotAllowed, server.ErrCodeMethodNotAllowed,
			"Method not allowed", false, map[string]interface{}{
				"method": r.Method,
			})
		return
	}

	q, err := ParseQuery(r)
	if err != nil {
		server.WriteError(w, r, http.StatusBadRequest, server.ErrCodeInvalidRequest,
			"Invalid recipe query", false, map[string]interface{}{
				"error": err.Error(),
			})
		return
	}

	if q == nil {
		server.WriteError(w, r, http.StatusBadRequest, server.ErrCodeInvalidRequest,
			"Recipe query cannot be empty", false, nil)
		return
	}

	slog.Debug("query",
		"os", q.Os.String(),
		"os_version", versionString(q.OsVersion),
		"kernel", versionString(q.Kernel),
		"service", q.Service.String(),
		"k8s", versionString(q.K8s),
		"gpu", q.GPU.String(),
		"intent", q.Intent.String(),
	)

	resp, err := b.Build(r.Context(), q)
	if err != nil {
		server.WriteError(w, r, http.StatusInternalServerError, server.ErrCodeInternalError,
			"Failed to build recipe", true, map[string]interface{}{
				"error": err.Error(),
			})
		return
	}

	if resp.Request.IsEmpty() {
		slog.Debug("stripping empty request from recipe response")
		resp.Request = nil
	}

	// Set caching headers
	w.Header().Set("Cache-Control", fmt.Sprintf("public, max-age=%d", recipeCacheTTLInSec))

	serializer.RespondJSON(w, http.StatusOK, resp)
}

// versionString returns the string representation of a version pointer,
// or "nil" if the pointer is nil.
func versionString(v *version.Version) string {
	if v == nil {
		return "nil"
	}
	return v.String()
}
