package internal

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	"github.com/NVIDIA/cloud-native-stack/pkg/bundler/config"
	"github.com/NVIDIA/cloud-native-stack/pkg/bundler/result"
	"github.com/NVIDIA/cloud-native-stack/pkg/bundler/types"
)

// BaseBundler provides common functionality for bundler implementations.
// Bundlers can use this to reuse standard operations and reduce boilerplate.
//
// Thread-safety: BaseBundler is safe for use by a single bundler instance.
// Do not share BaseBundler instances between concurrent bundler executions.
type BaseBundler struct {
	Config *config.Config
	Result *result.Result
}

// NewBaseBundler creates a new base bundler helper.
func NewBaseBundler(cfg *config.Config, bundlerType types.BundleType) *BaseBundler {
	if cfg == nil {
		cfg = config.NewConfig()
	}
	return &BaseBundler{
		Config: cfg,
		Result: result.New(bundlerType),
	}
}

// BundleDirectories holds the standard bundle directory structure.
type BundleDirectories struct {
	Root      string
	Scripts   string
	Manifests string
}

// CreateBundleDir creates the standard bundle directory structure.
// Returns the bundle directories for easy access to each subdirectory.
func (b *BaseBundler) CreateBundleDir(outputDir, bundleName string) (BundleDirectories, error) {
	bundleDir := filepath.Join(outputDir, bundleName)

	dirs := BundleDirectories{
		Root:      bundleDir,
		Scripts:   filepath.Join(bundleDir, "scripts"),
		Manifests: filepath.Join(bundleDir, "manifests"),
	}

	for _, dir := range []string{dirs.Root, dirs.Scripts, dirs.Manifests} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return dirs, fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	slog.Debug("bundle directory structure created",
		"bundle", bundleName,
		"root", dirs.Root,
	)

	return dirs, nil
}

// WriteFile writes content to a file and tracks it in the result.
// The file is created with the specified permissions and automatically
// added to the result's file list with its size.
func (b *BaseBundler) WriteFile(path string, content []byte, perm os.FileMode) error {
	if err := os.WriteFile(path, content, perm); err != nil {
		return fmt.Errorf("failed to write %s: %w", path, err)
	}

	b.Result.AddFile(path, int64(len(content)))

	slog.Debug("file written",
		"path", path,
		"size_bytes", len(content),
		"permissions", perm,
	)

	return nil
}

// WriteFileString writes string content to a file.
// This is a convenience wrapper around WriteFile for string content.
func (b *BaseBundler) WriteFileString(path, content string, perm os.FileMode) error {
	return b.WriteFile(path, []byte(content), perm)
}

// RenderTemplate renders a template with the given data.
// The template is parsed and executed with the provided data structure.
// Returns the rendered content as a string.
func (b *BaseBundler) RenderTemplate(tmplContent, name string, data interface{}) (string, error) {
	tmpl, err := template.New(name).Parse(tmplContent)
	if err != nil {
		return "", fmt.Errorf("failed to parse template %s: %w", name, err)
	}

	var buf strings.Builder
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute template %s: %w", name, err)
	}

	return buf.String(), nil
}

// RenderAndWriteTemplate renders a template and writes it to a file.
// This combines RenderTemplate and WriteFile for convenience.
func (b *BaseBundler) RenderAndWriteTemplate(tmplContent, name, outputPath string, data interface{}, perm os.FileMode) error {
	content, err := b.RenderTemplate(tmplContent, name, data)
	if err != nil {
		return err
	}

	return b.WriteFileString(outputPath, content, perm)
}

// GenerateChecksums creates a checksums.txt file for all generated files.
// The checksum file contains SHA256 hashes for verification of bundle integrity.
// Each line follows the format: "<hash>  <relative-path>"
func (b *BaseBundler) GenerateChecksums(ctx context.Context, bundleDir string) error {
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("context cancelled: %w", err)
	}

	checksums := make([]string, 0, len(b.Result.Files))

	for _, file := range b.Result.Files {
		data, err := os.ReadFile(file)
		if err != nil {
			return fmt.Errorf("failed to read %s for checksum: %w", file, err)
		}

		hash := sha256.Sum256(data)
		relPath, err := filepath.Rel(bundleDir, file)
		if err != nil {
			// If relative path fails, use absolute path
			relPath = file
		}

		checksums = append(checksums, fmt.Sprintf("%s  %s", hex.EncodeToString(hash[:]), relPath))
	}

	checksumPath := filepath.Join(bundleDir, "checksums.txt")
	content := strings.Join(checksums, "\n") + "\n"

	if err := b.WriteFileString(checksumPath, content, 0644); err != nil {
		return fmt.Errorf("failed to write checksums: %w", err)
	}

	slog.Debug("checksums generated",
		"file_count", len(checksums),
		"path", checksumPath,
	)

	return nil
}

// MakeExecutable changes file permissions to make a file executable.
// This is typically used for shell scripts after writing them.
func (b *BaseBundler) MakeExecutable(path string) error {
	if err := os.Chmod(path, 0755); err != nil {
		b.Result.AddError(fmt.Errorf("failed to make %s executable: %w", filepath.Base(path), err))
		return err
	}

	slog.Debug("file made executable", "path", path)
	return nil
}

// Finalize marks the bundler as successful and updates metrics.
// This should be called at the end of a successful bundle generation.
// It updates the result duration and marks success.
// Note: Bundlers should record their own Prometheus metrics after calling this.
func (b *BaseBundler) Finalize(start time.Time) {
	b.Result.Duration = time.Since(start)
	b.Result.MarkSuccess()

	slog.Debug("bundle generation finalized",
		"type", b.Result.Type,
		"files", len(b.Result.Files),
		"size_bytes", b.Result.Size,
		"duration", b.Result.Duration.Round(time.Millisecond),
	)
}

// CheckContext checks if the context has been canceled.
// This should be called periodically during long-running operations
// to allow for graceful cancellation.
func (b *BaseBundler) CheckContext(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		return nil
	}
}

// AddError adds a non-fatal error to the result.
// These errors are collected but do not stop bundle generation.
func (b *BaseBundler) AddError(err error) {
	if err != nil {
		b.Result.AddError(err)
		slog.Warn("non-fatal error during bundle generation",
			"type", b.Result.Type,
			"error", err,
		)
	}
}

const (
	//
	bundlerVersionKey       = "bundler_version"
	recipeBundlerVersionKey = "recipe-version"
)

// GetBundlerVersion retrieves the bundler version from the config map.
func GetBundlerVersion(m map[string]string) string {
	if v, ok := m[bundlerVersionKey]; ok {
		return v
	}
	return "unknown"
}

// GetRecipeBundlerVersion retrieves the bundler version from the recipe config map.
func GetRecipeBundlerVersion(m map[string]string) string {
	if v, ok := m[recipeBundlerVersionKey]; ok {
		return v
	}
	return "unknown"
}

// BuildBaseConfigMap creates a configuration map with common bundler settings.
// Returns a map containing namespace, helm_repository, helm_chart_version,
// and all custom labels/annotations with appropriate prefixes.
// Bundlers can extend this map with their specific values.
func (b *BaseBundler) BuildBaseConfigMap() map[string]string {
	config := make(map[string]string)

	config[bundlerVersionKey] = b.Config.Version()

	config["namespace"] = b.Config.Namespace()
	config["helm_repository"] = b.Config.HelmRepository()
	config["helm_chart_version"] = b.Config.HelmChartVersion()

	// Add custom labels with "label_" prefix
	for k, v := range b.Config.CustomLabels() {
		config["label_"+k] = v
	}

	// Add custom annotations with "annotation_" prefix
	for k, v := range b.Config.CustomAnnotations() {
		config["annotation_"+k] = v
	}

	return config
}

// TemplateFunc is a function that retrieves templates by name.
// Returns the template content and whether it was found.
type TemplateFunc func(name string) (string, bool)

// GenerateFileFromTemplate is a convenience method that combines template retrieval,
// rendering, and file writing in one call. This reduces boilerplate in bundler
// implementations by handling the common pattern of:
// 1. Get template by name
// 2. Check if template exists
// 3. Render template with data
// 4. Write rendered content to file
//
// Example usage:
//
//	err := b.GenerateFileFromTemplate(ctx, GetTemplate, "values.yaml",
//	    filepath.Join(dir, "values.yaml"), data, 0644)
func (b *BaseBundler) GenerateFileFromTemplate(ctx context.Context, getTemplate TemplateFunc,
	templateName, outputPath string, data interface{}, perm os.FileMode) error {

	if err := b.CheckContext(ctx); err != nil {
		return err
	}

	tmpl, ok := getTemplate(templateName)
	if !ok {
		return fmt.Errorf("%s template not found", templateName)
	}

	return b.RenderAndWriteTemplate(tmpl, templateName, outputPath, data, perm)
}
