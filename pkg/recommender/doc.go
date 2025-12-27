// Package recommender analyzes system snapshots and generates tailored configuration recommendations.
//
// # Overview
//
// The recommender package processes system snapshots captured by the snapshotter package,
// extracts relevant configuration parameters (OS, kernel, Kubernetes version, GPU type, etc.),
// and uses the recipe builder to generate optimized configuration recipes based on workload intent.
//
// # Core Concepts
//
// ConfigRecommender: Main service that coordinates snapshot analysis and recipe generation.
// The recommender extracts structured queries from snapshot measurements and delegates
// to the recipe builder for configuration matching.
//
// Query Extraction: Parses snapshot measurements to identify:
//   - Operating system family and version
//   - Kernel version (with vendor-specific handling)
//   - Kubernetes service provider (EKS, GKE, AKS, OKE, self-managed)
//   - Kubernetes version (with vendor-specific formats)
//   - GPU model
//   - Workload intent (training, inference)
//
// # Workload Intent
//
// Intent types guide recommendation selection:
//   - training: Optimizes for ML training workloads (throughput, multi-GPU)
//   - inference: Optimizes for inference workloads (latency, efficiency)
//   - any: Generic recommendations applicable to all workloads
//
// # Usage
//
// Basic recommendation generation:
//
//	recommender := recommender.New(
//	    recommender.WithVersion("v1.0.0"),
//	)
//
//	ctx := context.Background()
//	snapshot := // ... obtained from snapshotter
//
//	recipe, err := recommender.Recommend(ctx, recipe.IntentTraining, snapshot)
//	if err != nil {
//	    log.Fatalf("recommendation failed: %v", err)
//	}
//
//	// recipe contains optimized configuration settings
//	fmt.Printf("Recipe: %+v\n", recipe)
//
// With context timeout:
//
//	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
//	defer cancel()
//
//	recipe, err := recommender.Recommend(ctx, recipe.IntentInference, snapshot)
//	if err != nil {
//	    log.Fatalf("recommendation failed: %v", err)
//	}
//
// # Query Extraction Logic
//
// The recommender parses snapshot measurements to extract:
//
// Kubernetes Service Provider:
//   - Detects cloud provider from node providerID
//   - Maps: aws→EKS, gce→GKE, azure→AKS, oci→OKE
//   - Falls back to "self-managed" if no provider detected
//
// Kubernetes Version:
//   - Extracts from server version in K8s measurements
//   - Handles vendor-specific formats: "v1.33.5-eks-3025e55" → "1.33"
//   - Preserves major.minor precision for matching
//
// Kernel Version:
//   - Extracts from node measurements
//   - Handles vendor suffixes: "6.8.0-1028-aws" → "6.8"
//   - Used for kernel-specific optimizations
//
// Operating System:
//   - Identifies OS family and version from release measurements
//   - Maps VERSION_ID to version (e.g., "24.04")
//   - Supports Ubuntu, RHEL, COS
//
// GPU Detection:
//   - Extracts GPU model from GPU measurements
//   - Normalizes names: "NVIDIA H100 80GB HBM3" → "H100"
//   - Used for GPU-specific driver and operator settings
//
// # Recipe Generation
//
// Once a query is extracted, the recommender:
//
// 1. Validates all extracted parameters
// 2. Constructs a recipe.Query with include_context=true
// 3. Delegates to recipe.Builder for configuration matching
// 4. Returns a Recipe with base settings + overlays
//
// The Recipe contains:
//   - Metadata (version, creation time, matching criteria)
//   - Base measurements (common configuration)
//   - Overlay measurements (intent-specific optimizations)
//
// # Error Handling
//
// The recommender returns errors when:
//   - Snapshot is nil or empty
//   - Intent is invalid
//   - Required measurements are missing
//   - Query extraction fails
//   - Recipe building fails
//   - Context is canceled
//
// Callers should handle these errors and provide appropriate feedback.
//
// # Observability
//
// The recommender exports Prometheus metrics:
//   - recommend_generate_duration_seconds: Time to generate recommendations
//
// Structured logs are emitted at debug level for:
//   - Extracted query parameters
//   - Recipe matching details
//   - Error conditions
//
// # Integration
//
// The recommender is used by:
//   - pkg/cli - CLI recommend command
//   - pkg/server - API recommendation endpoints
//
// It depends on:
//   - pkg/snapshotter - Snapshot input
//   - pkg/recipe - Recipe building
//   - pkg/measurement - Data structures
package recommender
