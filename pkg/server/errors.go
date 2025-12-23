package server

import (
	"net/http"
	"time"

	"github.com/NVIDIA/cloud-native-stack/pkg/serializers"
	"github.com/google/uuid"
)

// Error codes as constants
const (
	ErrCodeRateLimitExceeded  = "RATE_LIMIT_EXCEEDED"
	ErrCodeInternalError      = "INTERNAL_ERROR"
	ErrCodeServiceUnavailable = "SERVICE_UNAVAILABLE"
	ErrCodeInvalidRequest     = "INVALID_REQUEST"
	ErrCodeMethodNotAllowed   = "METHOD_NOT_ALLOWED"
)

// writeError writes error response
func (s *Server) writeError(w http.ResponseWriter, r *http.Request, statusCode int,
	code, message string, retryable bool, details map[string]interface{}) {

	requestID, _ := r.Context().Value(contextKeyRequestID).(string)
	if requestID == "" {
		requestID = uuid.New().String()
	}

	errResp := ErrorResponse{
		Code:      code,
		Message:   message,
		Details:   details,
		RequestID: requestID,
		Timestamp: time.Now().UTC(),
		Retryable: retryable,
	}

	serializers.RespondJSON(w, statusCode, errResp)
}
