package server

import "net/http"

// responseWriter wraps http.ResponseWriter to track response status and prevent
// writing headers after the body has been written. This ensures proper HTTP semantics
// and helps catch middleware bugs where headers are set too late.
type responseWriter struct {
	http.ResponseWriter
	statusCode int
	written    bool
}

// newResponseWriter creates a new responseWriter that wraps the provided http.ResponseWriter.
func newResponseWriter(w http.ResponseWriter) *responseWriter {
	return &responseWriter{
		ResponseWriter: w,
		statusCode:     http.StatusOK,
		written:        false,
	}
}

// WriteHeader writes the HTTP status code. It only writes the first call,
// subsequent calls are ignored to prevent duplicate header writes.
func (rw *responseWriter) WriteHeader(statusCode int) {
	if rw.written {
		return // Prevent duplicate writes
	}
	rw.statusCode = statusCode
	rw.ResponseWriter.WriteHeader(statusCode)
	rw.written = true
}

// Write writes the response body. If WriteHeader hasn't been called,
// it automatically calls it with http.StatusOK.
func (rw *responseWriter) Write(b []byte) (int, error) {
	if !rw.written {
		rw.WriteHeader(http.StatusOK)
	}
	return rw.ResponseWriter.Write(b)
}

// Status returns the HTTP status code that was written.
func (rw *responseWriter) Status() int {
	return rw.statusCode
}
