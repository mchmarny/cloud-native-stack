package recipe

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// Recipe generation metrics
	recipeBuiltDuration = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "eidos_recipe_build_duration_seconds",
			Help:    "Duration of recipe generation in seconds",
			Buckets: []float64{1, 5, 10, 30, 60, 120, 300},
		},
	)

	// Recipe metadata cache metrics
	recipeCacheHits = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "eidos_recipe_cache_hits_total",
			Help: "Total number of recipe metadata cache hits",
		},
	)
	recipeCacheMisses = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "eidos_recipe_cache_misses_total",
			Help: "Total number of recipe metadata cache misses (initial loads)",
		},
	)
)
