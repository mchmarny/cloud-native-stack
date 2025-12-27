// Package recipe provides configuration recipe generation based on system parameters.
//
// # Overview
//
// The recipe package generates tailored configuration recommendations for GPU-accelerated
// Kubernetes clusters. It uses a base-plus-overlay model where base configurations are
// enhanced with intent-specific overlays based on matching query parameters.
//
// # Core Types
//
// Query: Specifies target environment parameters
//
//	type Query struct {
//	    Os             StringOpt  // Ubuntu, RHEL, ALL
//	    OsVersion      StringOpt  // 24.04, 22.04, ALL
//	    Kernel         StringOpt  // 6.8, 5.15, ALL
//	    Service        StringOpt  // eks, gke, aks, ALL
//	    K8s            StringOpt  // 1.33, 1.32, ALL
//	    GPU            StringOpt  // H100, GB200, ALL
//	    Intent         IntentType // training, inference, any
//	    IncludeContext bool       // Include context metadata
//	}
//
// Recipe: Generated configuration result
//
//	type Recipe struct {
//	    Header                            // API version, kind, metadata
//	    Request      *Query               // Input query
//	    MatchedRules []string             // Applied overlay rules
//	    Measurements []*measurement.Measurement // Configuration data
//	}
//
// Builder: Generates recipes from queries
//
//	type Builder struct {
//	    Version string  // Builder version for tracking
//	}
//
// # Intent Types
//
// Intent guides optimization strategy:
//   - IntentTraining: ML training workloads (throughput, multi-GPU)
//   - IntentInference: Inference workloads (latency, efficiency)
//   - IntentAny: Generic optimizations for all workloads
//
// # Usage
//
// Basic recipe generation:
//
//	query := &recipe.Query{
//	    Os:             recipe.NewStringOpt("Ubuntu"),
//	    OsVersion:      recipe.NewStringOpt("24.04"),
//	    Service:        recipe.NewStringOpt("eks"),
//	    GPU:            recipe.NewStringOpt("H100"),
//	    Intent:         recipe.IntentTraining,
//	    IncludeContext: true,
//	}
//
//	ctx := context.Background()
//	recipe, err := recipe.BuildRecipe(ctx, query)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	fmt.Printf("Matched rules: %v\n", recipe.MatchedRules)
//	for _, m := range recipe.Measurements {
//	    fmt.Printf("Type: %s, Subtypes: %d\n", m.Type, len(m.Subtypes))
//	}
//
// Custom builder with version:
//
//	builder := recipe.NewBuilder(
//	    recipe.WithVersion("v1.0.0"),
//	)
//
//	recipe, err := builder.Build(ctx, query)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
// Query with wildcard matching:
//
//	query := &recipe.Query{
//	    Os:        recipe.NewStringOpt("ALL"),  // Matches any OS
//	    Service:   recipe.NewStringOpt("eks"),
//	    GPU:       recipe.NewStringOpt("ALL"),  // Matches any GPU
//	    Intent:    recipe.IntentInference,
//	}
//
// # Query Matching
//
// Queries use a wildcard-based matching system:
//
// Exact Match:
//   - Query: Os="Ubuntu", OsVersion="24.04"
//   - Matches: Overlay with Os="Ubuntu", OsVersion="24.04"
//   - Does not match: OsVersion="22.04" or Os="RHEL"
//
// Wildcard Match:
//   - Query: Os="Ubuntu", OsVersion="ALL"
//   - Matches: Any OsVersion with Os="Ubuntu"
//
// Priority:
//   - More specific overlays take precedence
//   - Multiple matching overlays are applied in order
//   - Later overlays can override earlier ones
//
// # Base-Overlay Model
//
// Recipe generation follows this process:
//
// 1. Load base measurements (common to all configs)
// 2. Find matching overlays based on query
// 3. Merge overlay measurements into base (overlay wins on conflicts)
// 4. Return Recipe with combined measurements
//
// Store structure:
//
//	base:
//	  - type: K8s
//	    subtypes:
//	      - subtype: cluster
//	        data:
//	          common-setting: value
//	overlays:
//	  - key:
//	      gpu: H100
//	      intent: training
//	    types:
//	      - type: K8s
//	        subtypes:
//	          - subtype: cluster
//	            data:
//	              training-specific-setting: value
//
// # Version Handling
//
// Versions support semantic versioning with flexible precision:
//   - v1.33.5 → matches 1.33.5 exactly
//   - v1.33 → matches any 1.33.x
//   - v1 → matches any 1.x.x
//
// See pkg/recipe/version for version parsing and comparison details.
//
// # Context Metadata
//
// When Query.IncludeContext is true, measurements include context metadata:
//
//	measurements:
//	  - type: K8s
//	    subtypes:
//	      - subtype: cluster
//	        data:
//	          version: 1.33
//	        context:
//	          source: "base"
//	          applied: "2025-01-15T10:30:00Z"
//
// Context helps understand where settings originated (base vs overlays).
//
// # Error Handling
//
// Builder.Build() returns errors when:
//   - Query is nil
//   - Recipe data cannot be loaded
//   - Measurements cannot be merged
//
// Query validation errors occur when:
//   - Intent is invalid
//   - Version format is invalid
//
// # Data Source
//
// Recipe data is embedded at build time from:
//   - recipe/data/data-v1.yaml
//
// The store is loaded once and cached for all subsequent requests (singleton pattern).
//
// # Observability
//
// The recipe builder exports Prometheus metrics:
//   - recipe_built_duration_seconds: Time to build recipe
//   - recipe_rule_match_total{status}: Rule matching statistics
//
// # Integration
//
// The recipe package is used by:
//   - pkg/cli - recipe command
//   - pkg/recommender - Snapshot-based recommendations
//   - pkg/server - API recipe endpoints
//
// It depends on:
//   - pkg/measurement - Data structures
//   - pkg/recipe/version - Version parsing
//   - pkg/recipe/header - Common header types
//
// # Subpackages
//
//   - recipe/version - Semantic version parsing with flexible precision
//   - recipe/header - Common header structures for API resources
package recipe
