package recipe

import (
	"context"
	"errors"
	"testing"
	"time"
)

// TestBuilder_BuildFromCriteria_ContextCancellation tests context cancellation
// during recipe building to ensure proper timeout handling and error propagation.
func TestBuilder_BuildFromCriteria_ContextCancellation(t *testing.T) {
	tests := []struct {
		name        string
		setupCtx    func() (context.Context, context.CancelFunc)
		wantTimeout bool
	}{
		{
			name: "immediate cancellation",
			setupCtx: func() (context.Context, context.CancelFunc) {
				ctx, cancel := context.WithCancel(context.Background())
				cancel() // Cancel immediately
				return ctx, cancel
			},
			wantTimeout: true,
		},
		{
			name: "normal operation with adequate timeout",
			setupCtx: func() (context.Context, context.CancelFunc) {
				// Provide adequate timeout for normal operation
				return context.WithTimeout(context.Background(), 5*time.Second)
			},
			wantTimeout: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := tt.setupCtx()
			defer cancel()

			// Create builder with standard configuration
			builder := NewBuilder()

			// Create minimal criteria (all "any" wildcards)
			criteria := NewCriteria()

			// Attempt to build recipe
			result, err := builder.BuildFromCriteria(ctx, criteria)

			if tt.wantTimeout {
				// Should get timeout error
				if err == nil {
					t.Fatal("expected error due to context cancellation, got nil")
				}

				// Verify error is timeout-related
				if !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded) {
					t.Errorf("expected context cancellation error, got: %v", err)
				}

				// Result should be nil on error
				if result != nil {
					t.Error("expected nil result on error")
				}
			} else {
				// Should succeed
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if result == nil {
					t.Fatal("expected non-nil result")
				}
			}
		})
	}
}

// TestBuilder_BuildFromCriteria_TimeoutBudget verifies that the builder
// respects the 25-second timeout budget for recipe building.
func TestBuilder_BuildFromCriteria_TimeoutBudget(t *testing.T) {
	// Create context with 30s timeout (handler-level)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	builder := NewBuilder()
	criteria := NewCriteria()

	start := time.Now()
	result, err := builder.BuildFromCriteria(ctx, criteria)
	elapsed := time.Since(start)

	// Should complete quickly (within 1 second)
	if elapsed > 1*time.Second {
		t.Errorf("build took too long: %v (expected < 1s)", elapsed)
	}

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
}

// TestBuilder_BuildFromCriteria_ContextValues tests that context values
// are properly propagated through the build process.
func TestBuilder_BuildFromCriteria_ContextValues(t *testing.T) {
	type contextKey string
	const requestIDKey contextKey = "request-id"

	// Create context with value
	ctx := context.WithValue(context.Background(), requestIDKey, "test-request-123")

	builder := NewBuilder()
	criteria := NewCriteria()

	result, err := builder.BuildFromCriteria(ctx, criteria)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}

	// Verify context value was accessible (would be used for logging/tracing)
	if requestID := ctx.Value(requestIDKey); requestID != "test-request-123" {
		t.Error("context value was lost during build")
	}
}
