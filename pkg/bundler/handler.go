package bundler

import (
	"archive/zip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"

	"github.com/NVIDIA/cloud-native-stack/pkg/bundler/config"
	"github.com/NVIDIA/cloud-native-stack/pkg/bundler/registry"
	"github.com/NVIDIA/cloud-native-stack/pkg/bundler/result"
	"github.com/NVIDIA/cloud-native-stack/pkg/bundler/types"
	deployerRegistry "github.com/NVIDIA/cloud-native-stack/pkg/deployer/registry"
	deployerTypes "github.com/NVIDIA/cloud-native-stack/pkg/deployer/types"
	cnserrors "github.com/NVIDIA/cloud-native-stack/pkg/errors"
	"github.com/NVIDIA/cloud-native-stack/pkg/recipe"
	"github.com/NVIDIA/cloud-native-stack/pkg/server"
	"github.com/NVIDIA/cloud-native-stack/pkg/snapshotter"
)

const (
	// DefaultBundleTimeout is the timeout for bundle generation.
	// Bundle generation involves parallel file I/O and template rendering.
	DefaultBundleTimeout = 60 * time.Second
)

// HandleBundles processes bundle generation requests.
// It accepts a POST request with a JSON body containing the recipe (RecipeResult).
// Supports query parameters:
//   - bundlers: Comma-delimited list of bundler types (e.g., "gpu-operator,network-operator")
//   - set: Value overrides in format "bundler:path.to.field=value" (can be repeated)
//   - system-node-selector: Node selectors for system components in format "key=value" (can be repeated)
//   - system-node-toleration: Tolerations for system components in format "key=value:effect" (can be repeated)
//   - accelerated-node-selector: Node selectors for GPU nodes in format "key=value" (can be repeated)
//   - accelerated-node-toleration: Tolerations for GPU nodes in format "key=value:effect" (can be repeated)
//   - deployer: Deployment method (script, argocd, flux, default: script)
//
// If no bundlers are specified, all registered bundlers are executed.
// The response is a zip archive containing all generated bundles.
//
// Example:
//
//	POST /v1/bundle?bundlers=gpu-operator&deployer=argocd&set=gpuoperator:gds.enabled=true
//	Content-Type: application/json
//	Body: { "apiVersion": "cns.nvidia.com/v1alpha1", "kind": "Recipe", ... }
func (b *DefaultBundler) HandleBundles(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		server.WriteError(w, r, http.StatusMethodNotAllowed, cnserrors.ErrCodeMethodNotAllowed,
			"Method not allowed", false, map[string]interface{}{
				"method": r.Method,
			})
		return
	}

	// Add request-scoped timeout
	ctx, cancel := context.WithTimeout(r.Context(), DefaultBundleTimeout)
	defer cancel()

	// Parse all query parameters
	params, err := parseQueryParams(r)
	if err != nil {
		server.WriteErrorFromErr(w, r, err, "Invalid query parameters", nil)
		return
	}

	// Parse request body directly as RecipeResult
	var recipeResult recipe.RecipeResult
	err = json.NewDecoder(r.Body).Decode(&recipeResult)
	if err != nil {
		server.WriteError(w, r, http.StatusBadRequest, cnserrors.ErrCodeInvalidRequest,
			"Invalid request body", false, map[string]interface{}{
				"error": err.Error(),
			})
		return
	}

	// Validate recipe has component references
	if len(recipeResult.ComponentRefs) == 0 {
		server.WriteError(w, r, http.StatusBadRequest, cnserrors.ErrCodeInvalidRequest,
			"Recipe must contain at least one component reference", false, nil)
		return
	}

	slog.Debug("bundle request received",
		"components", len(recipeResult.ComponentRefs),
		"bundlers", params.bundlerNames,
		"deployer", params.deployerType,
		"value_overrides", len(params.valueOverrides),
		"system_node_selectors", len(params.systemNodeSelector),
		"accelerated_node_selectors", len(params.acceleratedNodeSelector),
	)

	// Parse bundler types
	bundlerTypes, err := parseBundlerTypes(params.bundlerNames)
	if err != nil {
		server.WriteErrorFromErr(w, r, err, "Invalid bundler type", nil)
		return
	}

	// Create temporary directory for bundle output
	tempDir, err := os.MkdirTemp("", "cns-bundle-*")
	if err != nil {
		server.WriteError(w, r, http.StatusInternalServerError, cnserrors.ErrCodeInternal,
			"Failed to create temporary directory", true, nil)
		return
	}
	defer os.RemoveAll(tempDir) // Clean up on exit

	// Create bundler registry from global with config
	reg := registry.NewFromGlobal(
		config.NewConfig(
			config.WithValueOverrides(params.valueOverrides),
			config.WithSystemNodeSelector(params.systemNodeSelector),
			config.WithSystemNodeTolerations(params.systemNodeTolerations),
			config.WithAcceleratedNodeSelector(params.acceleratedNodeSelector),
			config.WithAcceleratedNodeTolerations(params.acceleratedNodeTolerations),
		),
	)

	// Create a new bundler with specified types (or use all if empty)
	bundler, err := New(
		WithBundlerTypes(bundlerTypes),
		WithFailFast(false), // Collect all errors
		WithRegistry(reg),
		WithDeployer(params.deployerType),
	)
	if err != nil {
		server.WriteError(w, r, http.StatusInternalServerError, cnserrors.ErrCodeInternal,
			"Failed to create bundler", true, map[string]interface{}{
				"error": err.Error(),
			})
		return
	}

	// Generate bundles
	output, err := bundler.Make(ctx, &recipeResult, tempDir)
	if err != nil {
		server.WriteErrorFromErr(w, r, err, "Failed to generate bundles", nil)
		return
	}

	// Check for bundle errors
	if output.HasErrors() {
		errorDetails := make([]map[string]interface{}, 0, len(output.Errors))
		for _, be := range output.Errors {
			errorDetails = append(errorDetails, map[string]interface{}{
				"bundler": be.BundlerType,
				"error":   be.Error,
			})
		}
		server.WriteError(w, r, http.StatusInternalServerError, cnserrors.ErrCodeInternal,
			"Some bundlers failed", true, map[string]interface{}{
				"errors":        errorDetails,
				"success_count": output.SuccessCount(),
			})
		return
	}

	// Stream zip response
	if err := streamZipResponse(w, tempDir, output); err != nil {
		// Can't write error response if we've already started writing
		slog.Error("failed to stream zip response", "error", err)
		return
	}
}

// streamZipResponse creates a zip archive from the output directory and streams it to the response.
func streamZipResponse(w http.ResponseWriter, dir string, output *result.Output) error {
	// Set response headers before writing body
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", "attachment; filename=\"bundles.zip\"")
	w.Header().Set("X-Bundle-Files", strconv.Itoa(output.TotalFiles))
	w.Header().Set("X-Bundle-Size", strconv.FormatInt(output.TotalSize, 10))
	w.Header().Set("X-Bundle-Duration", output.TotalDuration.String())

	// Create zip writer directly to response
	zw := zip.NewWriter(w)
	defer zw.Close()

	// Walk the directory and add all files to zip
	return filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return fmt.Errorf("walk error: %w", err)
		}

		// Skip the root directory itself
		if path == dir {
			return nil
		}

		// Get relative path for zip entry
		relPath, err := filepath.Rel(dir, path)
		if err != nil {
			return fmt.Errorf("failed to get relative path: %w", err)
		}

		// Create zip file header
		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return fmt.Errorf("failed to create file header: %w", err)
		}
		header.Name = relPath

		// Preserve directory structure
		if info.IsDir() {
			header.Name += "/"
			_, headerErr := zw.CreateHeader(header)
			return headerErr
		}

		// Use deflate compression
		header.Method = zip.Deflate

		writer, err := zw.CreateHeader(header)
		if err != nil {
			return fmt.Errorf("failed to create zip entry: %w", err)
		}

		// Open and copy file content
		file, err := os.Open(path)
		if err != nil {
			return fmt.Errorf("failed to open file: %w", err)
		}
		defer file.Close()

		_, err = io.Copy(writer, file)
		if err != nil {
			return fmt.Errorf("failed to copy file content: %w", err)
		}

		return nil
	})
}

// deployerTypesToStrings converts deployer types to string slice for error messages
func deployerTypesToStrings(types []deployerTypes.DeployerType) []string {
	result := make([]string, len(types))
	for i, t := range types {
		result[i] = string(t)
	}
	return result
}

// bundleParams holds parsed query parameters for bundle generation
type bundleParams struct {
	bundlerNames               []string
	valueOverrides             map[string]map[string]string
	systemNodeSelector         map[string]string
	systemNodeTolerations      []corev1.Toleration
	acceleratedNodeSelector    map[string]string
	acceleratedNodeTolerations []corev1.Toleration
	deployerType               deployerTypes.DeployerType
}

// parseQueryParams extracts and validates all query parameters from the request
func parseQueryParams(r *http.Request) (*bundleParams, error) {
	query := r.URL.Query()
	params := &bundleParams{}

	// Parse bundlers
	bundlersParam := query.Get("bundlers")
	if bundlersParam != "" {
		params.bundlerNames = strings.Split(bundlersParam, ",")
	}

	// Parse value overrides
	var err error
	params.valueOverrides, err = config.ParseValueOverrides(query["set"])
	if err != nil {
		return nil, cnserrors.Wrap(cnserrors.ErrCodeInvalidRequest, "Invalid set parameter", err)
	}

	// Parse system node selectors
	params.systemNodeSelector, err = snapshotter.ParseNodeSelectors(query["system-node-selector"])
	if err != nil {
		return nil, cnserrors.Wrap(cnserrors.ErrCodeInvalidRequest, "Invalid system-node-selector", err)
	}

	// Parse accelerated node selectors
	params.acceleratedNodeSelector, err = snapshotter.ParseNodeSelectors(query["accelerated-node-selector"])
	if err != nil {
		return nil, cnserrors.Wrap(cnserrors.ErrCodeInvalidRequest, "Invalid accelerated-node-selector", err)
	}

	// Parse system node tolerations
	params.systemNodeTolerations, err = snapshotter.ParseTolerations(query["system-node-toleration"])
	if err != nil {
		return nil, cnserrors.Wrap(cnserrors.ErrCodeInvalidRequest, "Invalid system-node-toleration", err)
	}

	// Parse accelerated node tolerations
	params.acceleratedNodeTolerations, err = snapshotter.ParseTolerations(query["accelerated-node-toleration"])
	if err != nil {
		return nil, cnserrors.Wrap(cnserrors.ErrCodeInvalidRequest, "Invalid accelerated-node-toleration", err)
	}

	// Parse and validate deployer type
	deployerTypeStr := query.Get("deployer")
	if deployerTypeStr == "" {
		deployerTypeStr = string(deployerTypes.DeployerTypeScript) // Default
	}
	params.deployerType = deployerTypes.DeployerType(deployerTypeStr)
	if !params.deployerType.IsValid() {
		deployerReg := deployerRegistry.NewFromGlobal()
		registeredTypes := deployerReg.Types()
		err := cnserrors.New(cnserrors.ErrCodeInvalidRequest, "Invalid deployer parameter")
		err.Context = map[string]interface{}{
			"deployer": deployerTypeStr,
			"valid":    deployerTypesToStrings(registeredTypes),
		}
		return nil, err
	}

	return params, nil
}

// parseBundlerTypes parses and validates bundler type names
func parseBundlerTypes(bundlerNames []string) ([]types.BundleType, error) {
	bundlerTypes := make([]types.BundleType, 0, len(bundlerNames))
	for _, bt := range bundlerNames {
		bt = strings.TrimSpace(bt)
		if bt == "" {
			continue
		}
		parsed, err := types.ParseType(bt)
		if err != nil {
			return nil, cnserrors.WrapWithContext(
				cnserrors.ErrCodeInvalidRequest,
				"Invalid bundler type",
				err,
				map[string]interface{}{
					"bundler": bt,
					"valid":   types.SupportedBundleTypesAsStrings(),
				},
			)
		}
		bundlerTypes = append(bundlerTypes, parsed)
	}
	return bundlerTypes, nil
}
