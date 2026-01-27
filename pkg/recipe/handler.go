package recipe

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	cnserrors "github.com/NVIDIA/cloud-native-stack/pkg/errors"
	"github.com/NVIDIA/cloud-native-stack/pkg/serializer"
	"github.com/NVIDIA/cloud-native-stack/pkg/server"
)

const (
	// DefaultRecipeCacheTTL is the default cache duration for recipe responses.
	// 10 minutes balances freshness with reduced load on recipe store.
	// Recipe metadata rarely changes; longer TTL would be safe but reduces
	// visibility of updates during development.
	DefaultRecipeCacheTTL = 10 * time.Minute
)

var (
	// recipeCacheTTL can be overridden for testing or custom configurations
	recipeCacheTTL = DefaultRecipeCacheTTL
)

// HandleRecipes processes recipe requests using the criteria-based system.
// It supports GET requests with query parameters to specify recipe criteria.
// The response returns a RecipeResult with component references and constraints.
// Errors are handled and returned in a structured format.
func (b *Builder) HandleRecipes(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		server.WriteError(w, r, http.StatusMethodNotAllowed, cnserrors.ErrCodeMethodNotAllowed,
			"Method not allowed", false, map[string]interface{}{
				"method": r.Method,
			})
		return
	}

	// Add request-scoped timeout
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	criteria, err := ParseCriteriaFromRequest(r)
	if err != nil {
		server.WriteError(w, r, http.StatusBadRequest, cnserrors.ErrCodeInvalidRequest,
			"Invalid recipe criteria", false, map[string]interface{}{
				"error": err.Error(),
			})
		return
	}

	if criteria == nil {
		server.WriteError(w, r, http.StatusBadRequest, cnserrors.ErrCodeInvalidRequest,
			"Recipe criteria cannot be empty", false, nil)
		return
	}

	slog.Debug("criteria",
		"service", criteria.Service,
		"accelerator", criteria.Accelerator,
		"intent", criteria.Intent,
		"os", criteria.OS,
		"nodes", criteria.Nodes,
	)

	// Validate criteria against allowlists (if configured)
	if b.AllowLists != nil {
		if validateErr := b.AllowLists.ValidateCriteria(criteria); validateErr != nil {
			server.WriteErrorFromErr(w, r, validateErr, "Criteria value not allowed", nil)
			return
		}
	}

	result, err := b.BuildFromCriteria(ctx, criteria)
	if err != nil {
		server.WriteErrorFromErr(w, r, err, "Failed to build recipe", nil)
		return
	}

	// Set caching headers
	w.Header().Set("Cache-Control", fmt.Sprintf("public, max-age=%d", int(recipeCacheTTL.Seconds())))

	serializer.RespondJSON(w, http.StatusOK, result)
}
