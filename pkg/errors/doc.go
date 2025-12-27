// Package errors provides structured error types for better observability
// and programmatic error handling across the application.
//
// Example usage:
//
//	err := errors.WrapWithContext(
//	    errors.ErrCodeTimeout,
//	    "failed to collect GPU metrics",
//	    ctx.Err(),
//	    map[string]interface{}{
//	        "command": "nvidia-smi",
//	        "node": nodeName,
//	    },
//	)
package errors
