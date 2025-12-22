package server

import (
	"fmt"
	"net/http"
	"time"

	"github.com/NVIDIA/cloud-native-stack/pkg/measurement"
)

// Handler implementations

// handleGetRecommendations handles GET /v1/recommendations
func (s *Server) handleGetRecommendations(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.writeError(w, r, http.StatusMethodNotAllowed, ErrCodeMethodNotAllowed,
			"Method not allowed", false, nil)
		return
	}

	// Parse and validate query parameters
	req := s.parseRecommendationRequest(r)
	if err := s.validator.ValidateRecommendationRequest(req); err != nil {
		s.writeError(w, r, http.StatusBadRequest, ErrCodeInvalidParameter,
			err.Error(), false, map[string]interface{}{
				"request": req,
			})
		return
	}

	// Generate recommendation (stub implementation)
	resp := s.generateRecommendation(req)

	// Set cache headers
	w.Header().Set("Cache-Control", fmt.Sprintf("public, max-age=%d", s.config.CacheMaxAge))

	s.writeJSON(w, http.StatusOK, resp)
}

// Helper methods

// parseRecommendationRequest extracts query parameters into request struct
func (s *Server) parseRecommendationRequest(r *http.Request) *RecommendationRequest {
	q := r.URL.Query()

	req := &RecommendationRequest{
		OSFamily:    getQueryParamOrDefault(q, "osFamily"),
		OSVersion:   getQueryParamOrDefault(q, "osVersion"),
		Kernel:      getQueryParamOrDefault(q, "kernel"),
		Environment: getQueryParamOrDefault(q, "environment"),
		Kubernetes:  getQueryParamOrDefault(q, "kubernetes"),
		GPU:         getQueryParamOrDefault(q, "gpu"),
		Intent:      getQueryParamOrDefault(q, "intent"),
	}

	if pv := q.Get("payloadVersion"); pv != "" {
		req.PayloadVersionRequested = &pv
	}

	return req
}

// generateRecommendation creates a recommendation response (stub implementation)
func (s *Server) generateRecommendation(req *RecommendationRequest) RecommendationResponse {
	// TODO: Implement real recommendation logic
	version := "2025.12.0"

	return RecommendationResponse{
		Request:        *req,
		MatchedRuleID:  "default-rule",
		PayloadVersion: version,
		GeneratedAt:    time.Now().UTC(),
		Measurements: []*measurement.Measurement{
			{Type: "example",
				Subtypes: []measurement.Subtype{
					{
						Name: "sample",
						Data: map[string]measurement.Reading{
							"key1": measurement.Str("value1"),
							"key2": measurement.Int(42),
						},
					},
				},
			},
		},
	}
}
