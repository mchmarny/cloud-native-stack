package server

import (
	"fmt"
	"net/http"

	"github.com/NVIDIA/cloud-native-stack/pkg/recommendation"
	"github.com/NVIDIA/cloud-native-stack/pkg/serializers"
)

// handleRecommendations processes GET /v1/recommendations requests end-to-end, ensuring
// structured error responses consistent with the rest of the server surface.
func (s *Server) handleRecommendations(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		s.writeError(w, r, http.StatusMethodNotAllowed, ErrCodeMethodNotAllowed,
			"Method not allowed", false, map[string]interface{}{
				"method": r.Method,
			})
		return
	}

	q, err := recommendation.ParseQuery(r)
	if err != nil {
		s.writeError(w, r, http.StatusBadRequest, ErrCodeInvalidRequest,
			"Invalid recommendation query", false, map[string]interface{}{
				"error": err.Error(),
			})
		return
	}

	resp, err := s.recommendationBuilder.Build(q)
	if err != nil {
		s.writeError(w, r, http.StatusInternalServerError, ErrCodeInternalError,
			"Failed to build recommendation", true, map[string]interface{}{
				"error": err.Error(),
			})
		return
	}

	if ttl := s.recommendationBuilder.CacheTTL; ttl > 0 {
		w.Header().Set("Cache-Control", fmt.Sprintf("public, max-age=%d", int(ttl.Seconds())))
	} else {
		w.Header().Set("Cache-Control", "no-store")
	}

	serializers.RespondJSON(w, http.StatusOK, resp)
}
