package server

// contextKey is a custom type for context keys to avoid collisions
type contextKey string

const (
	// contextKeyRequestID is the context key for request ID
	contextKeyRequestID contextKey = "requestID"
	// contextKeyAPIVersion is the context key for API version
	contextKeyAPIVersion contextKey = "apiVersion"
)
