// Package api provides the HTTP API layer for the CNS Recipe Generation service.
//
// This package acts as a thin wrapper around the reusable pkg/server package,
// configuring it with application-specific routes and handlers. It exposes the
// recipe generation functionality (Step 2 of the three-stage workflow) via REST API.
//
// # Usage
//
// To start the API server:
//
//	package main
//
//	import (
//	    "log"
//	    "github.com/NVIDIA/cloud-native-stack/pkg/api"
//	)
//
//	func main() {
//	    if err := api.Serve(); err != nil {
//	        log.Fatalf("server error: %v", err)
//	    }
//	}
//
// # Architecture
//
// The API layer is responsible for:
//   - Configuring structured logging with application name and version
//   - Setting up route handlers (e.g., /v1/recipe)
//   - Delegating server lifecycle management to pkg/server
//
// The pkg/server package handles:
//   - HTTP server setup and graceful shutdown
//   - Middleware (rate limiting, logging, metrics, panic recovery)
//   - Health and readiness endpoints
//   - Prometheus metrics
//
// # Endpoints
//
// Application Endpoints (with rate limiting):
//   - GET /v1/recipe - Generate configuration recipe based on query parameters
//
// System Endpoints (no rate limiting):
//   - GET /health  - Health check (liveness probe)
//   - GET /ready   - Readiness check
//   - GET /metrics - Prometheus metrics
//
// # Query Parameters
//
// The /v1/recipe endpoint accepts:
//   - os: Operating system (ubuntu, cos, rhel, any)
//   - osv: OS version (e.g., 24.04)
//   - kernel: Kernel version (supports vendor suffixes)
//   - service: Kubernetes service (eks, gke, aks, self-managed, any)
//   - k8s: Kubernetes version (supports vendor suffixes)
//   - gpu: GPU type (h100, gb200, a100, l40, any)
//   - intent: Workload intent (training, inference, any)
//   - context: Include context metadata (true/false)
//
// # Configuration
//
// The server is configured via environment variables:
//   - PORT: HTTP server port (default: 8080)
//   - LOG_LEVEL: Logging level (debug, info, warn, error)
//
// Version information is set at build time using ldflags:
//
//	go build -ldflags="-X 'github.com/NVIDIA/cloud-native-stack/pkg/api.version=1.0.0'"
package api
