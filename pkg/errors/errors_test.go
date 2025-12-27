package errors

import (
	"errors"
	"testing"
)

func TestNew(t *testing.T) {
	err := New(ErrCodeNotFound, "resource not found")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err.Code != ErrCodeNotFound {
		t.Errorf("expected code %s, got %s", ErrCodeNotFound, err.Code)
	}
	if err.Message != "resource not found" {
		t.Errorf("expected message 'resource not found', got %s", err.Message)
	}
	if err.Cause != nil {
		t.Errorf("expected nil cause, got %v", err.Cause)
	}
}

func TestWrap(t *testing.T) {
	cause := errors.New("underlying error")
	err := Wrap(ErrCodeInternal, "operation failed", cause)

	if err.Code != ErrCodeInternal {
		t.Errorf("expected code %s, got %s", ErrCodeInternal, err.Code)
	}
	if !errors.Is(err, cause) {
		t.Errorf("expected cause to be wrapped")
	}
}

func TestWrapWithContext(t *testing.T) {
	cause := errors.New("timeout")
	ctx := map[string]interface{}{
		"command": "nvidia-smi",
		"node":    "node-1",
	}

	err := WrapWithContext(ErrCodeTimeout, "GPU collection failed", cause, ctx)

	if err.Code != ErrCodeTimeout {
		t.Errorf("expected code %s, got %s", ErrCodeTimeout, err.Code)
	}
	if err.Context == nil {
		t.Fatal("expected context to be set")
	}
	if err.Context["command"] != "nvidia-smi" {
		t.Errorf("expected command to be nvidia-smi")
	}
}

func TestError(t *testing.T) {
	tests := []struct {
		name     string
		err      *StructuredError
		expected string
	}{
		{
			name:     "error without cause",
			err:      New(ErrCodeNotFound, "not found"),
			expected: "[NOT_FOUND] not found",
		},
		{
			name:     "error with cause",
			err:      Wrap(ErrCodeInternal, "failed", errors.New("root cause")),
			expected: "[INTERNAL] failed: root cause",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.err.Error()
			if got != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, got)
			}
		})
	}
}

func TestUnwrap(t *testing.T) {
	cause := errors.New("root cause")
	err := Wrap(ErrCodeInternal, "wrapped", cause)

	unwrapped := err.Unwrap()
	if !errors.Is(unwrapped, cause) {
		t.Errorf("expected unwrapped error to be original cause")
	}

	if !errors.Is(err, cause) {
		t.Errorf("errors.Is should work with Unwrap")
	}
}

func TestErrorCodes(t *testing.T) {
	codes := []ErrorCode{
		ErrCodeNotFound,
		ErrCodeUnauthorized,
		ErrCodeTimeout,
		ErrCodeInternal,
		ErrCodeInvalidRequest,
		ErrCodeUnavailable,
	}

	for _, code := range codes {
		if string(code) == "" {
			t.Errorf("error code should not be empty: %v", code)
		}
	}
}
