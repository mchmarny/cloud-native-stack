package api

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/NVIDIA/cloud-native-stack/pkg/bundler"
	"github.com/NVIDIA/cloud-native-stack/pkg/logging"
	"github.com/NVIDIA/cloud-native-stack/pkg/recipe"
	"github.com/NVIDIA/cloud-native-stack/pkg/server"
)

const (
	name           = "cnsd"
	versionDefault = "dev"
)

var (
	// overridden during build with ldflags to reflect actual version info
	// e.g., -X "github.com/NVIDIA/cloud-native-stack/pkg/api.version=1.0.0"
	version = versionDefault
	commit  = "unknown"
	date    = "unknown"
)

// Serve starts the API server and blocks until shutdown.
// It configures logging, sets up routes, and handles graceful shutdown.
// Returns an error if the server fails to start or encounters a fatal error.
func Serve() error {
	ctx := context.Background()

	logging.SetDefaultStructuredLogger(name, version)
	slog.Debug("starting",
		"name", name,
		"version", version,
		"commit", commit,
		"date", date,
	)

	// Parse allowlists from environment variables
	allowLists, err := recipe.ParseAllowListsFromEnv()
	if err != nil {
		return fmt.Errorf("failed to parse allowlists from environment: %w", err)
	}

	if allowLists != nil {
		slog.Info("criteria allowlists configured",
			"accelerators", len(allowLists.Accelerators),
			"services", len(allowLists.Services),
			"intents", len(allowLists.Intents),
			"os_types", len(allowLists.OSTypes),
		)
		slog.Debug("criteria allowlists loaded",
			"accelerators", allowLists.AcceleratorStrings(),
			"services", allowLists.ServiceStrings(),
			"intents", allowLists.IntentStrings(),
			"os_types", allowLists.OSTypeStrings(),
		)
	}

	// Setup recipe handler
	rb := recipe.NewBuilder(
		recipe.WithVersion(version),
		recipe.WithAllowLists(allowLists),
	)

	// Setup bundle handler
	bb, err := bundler.New(
		bundler.WithAllowLists(allowLists),
	)
	if err != nil {
		return fmt.Errorf("failed to create bundler: %w", err)
	}

	r := map[string]http.HandlerFunc{
		"/v1/recipe": rb.HandleRecipes,
		"/v1/bundle": bb.HandleBundles,
	}

	// Create and run server
	s := server.New(
		server.WithName(name),
		server.WithVersion(version),
		server.WithHandler(r),
	)

	if err := s.Run(ctx); err != nil {
		slog.Error("server exited with error", "error", err)
		return err
	}

	return nil
}
