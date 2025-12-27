package errors

import "fmt"

// ErrorCode represents a structured error classification.
type ErrorCode string

const (
	// ErrCodeNotFound indicates a requested resource was not found.
	ErrCodeNotFound ErrorCode = "NOT_FOUND"
	// ErrCodeUnauthorized indicates authentication or authorization failure.
	ErrCodeUnauthorized ErrorCode = "UNAUTHORIZED"
	// ErrCodeTimeout indicates an operation exceeded its time limit.
	ErrCodeTimeout ErrorCode = "TIMEOUT"
	// ErrCodeInternal indicates an internal system error.
	ErrCodeInternal ErrorCode = "INTERNAL"
	// ErrCodeInvalidRequest indicates malformed or invalid input.
	ErrCodeInvalidRequest ErrorCode = "INVALID_REQUEST"
	// ErrCodeUnavailable indicates a service or resource is temporarily unavailable.
	ErrCodeUnavailable ErrorCode = "UNAVAILABLE"
)

// StructuredError provides structured error information for better observability.
// It includes an error code for programmatic handling, a human-readable message,
// the underlying cause, and optional context for debugging.
type StructuredError struct {
	Code    ErrorCode
	Message string
	Cause   error
	Context map[string]interface{}
}

// Error implements the error interface.
func (e *StructuredError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("[%s] %s: %v", e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

// Unwrap returns the underlying cause for errors.Is and errors.As support.
func (e *StructuredError) Unwrap() error {
	return e.Cause
}

// New creates a new StructuredError with the given code and message.
func New(code ErrorCode, message string) *StructuredError {
	return &StructuredError{
		Code:    code,
		Message: message,
	}
}

// Wrap wraps an existing error with additional context.
func Wrap(code ErrorCode, message string, cause error) *StructuredError {
	return &StructuredError{
		Code:    code,
		Message: message,
		Cause:   cause,
	}
}

// WrapWithContext wraps an error with additional context information.
func WrapWithContext(code ErrorCode, message string, cause error, context map[string]interface{}) *StructuredError {
	return &StructuredError{
		Code:    code,
		Message: message,
		Cause:   cause,
		Context: context,
	}
}
