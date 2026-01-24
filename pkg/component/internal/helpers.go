package internal

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"text/template"

	"gopkg.in/yaml.v3"

	"github.com/NVIDIA/cloud-native-stack/pkg/bundler/result"
)

// TemplateRenderer provides template rendering functionality for bundlers.
type TemplateRenderer struct {
	// templateGetter is a function that retrieves template content by name.
	templateGetter func(name string) (string, bool)
}

// NewTemplateRenderer creates a new template renderer with the given template getter.
func NewTemplateRenderer(getter func(name string) (string, bool)) *TemplateRenderer {
	return &TemplateRenderer{
		templateGetter: getter,
	}
}

// Render renders a template with the given data.
func (r *TemplateRenderer) Render(name string, data map[string]interface{}) (string, error) {
	tmplContent, ok := r.templateGetter(name)
	if !ok {
		return "", fmt.Errorf("template %s not found", name)
	}

	tmpl, err := template.New(name).Parse(tmplContent)
	if err != nil {
		return "", fmt.Errorf("failed to parse template %s: %w", name, err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute template %s: %w", name, err)
	}

	return buf.String(), nil
}

// FileWriter provides file writing functionality with result tracking.
type FileWriter struct {
	result *result.Result
}

// NewFileWriter creates a new file writer with the given result tracker.
func NewFileWriter(result *result.Result) *FileWriter {
	return &FileWriter{
		result: result,
	}
}

// WriteFile writes content to a file with the specified permissions and updates the result.
func (w *FileWriter) WriteFile(path string, content []byte, perm os.FileMode) error {
	if err := os.WriteFile(path, content, perm); err != nil {
		return fmt.Errorf("failed to write file %s: %w", path, err)
	}

	w.result.AddFile(path, int64(len(content)))

	slog.Debug("file written",
		"path", path,
		"size_bytes", len(content),
		"permissions", perm,
	)

	return nil
}

// WriteFileString writes string content to a file with the specified permissions.
func (w *FileWriter) WriteFileString(path, content string, perm os.FileMode) error {
	return w.WriteFile(path, []byte(content), perm)
}

// MakeExecutable changes file permissions to make it executable.
func (w *FileWriter) MakeExecutable(path string) error {
	if err := os.Chmod(path, 0755); err != nil {
		w.result.AddError(fmt.Errorf("failed to make %s executable: %w", filepath.Base(path), err))
		return err
	}
	return nil
}

// DirectoryManager provides directory management functionality.
type DirectoryManager struct{}

// NewDirectoryManager creates a new directory manager.
func NewDirectoryManager() *DirectoryManager {
	return &DirectoryManager{}
}

// CreateDirectories creates multiple directories with the specified permissions.
func (m *DirectoryManager) CreateDirectories(dirs []string, perm os.FileMode) error {
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, perm); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}
	return nil
}

// CreateBundleStructure creates the standard bundle directory structure.
// Returns the bundle root directory and subdirectories (scripts, manifests).
func (m *DirectoryManager) CreateBundleStructure(outputDir, bundleName string) (string, map[string]string, error) {
	bundleDir := filepath.Join(outputDir, bundleName)
	scriptsDir := filepath.Join(bundleDir, "scripts")
	manifestsDir := filepath.Join(bundleDir, "manifests")

	dirs := []string{bundleDir, scriptsDir, manifestsDir}
	if err := m.CreateDirectories(dirs, 0755); err != nil {
		return "", nil, err
	}

	subdirs := map[string]string{
		"root":      bundleDir,
		"scripts":   scriptsDir,
		"manifests": manifestsDir,
	}

	return bundleDir, subdirs, nil
}

// ContextChecker provides context cancellation checking.
type ContextChecker struct{}

// NewContextChecker creates a new context checker.
func NewContextChecker() *ContextChecker {
	return &ContextChecker{}
}

// Check checks if the context has been canceled.
func (c *ContextChecker) Check(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		return nil
	}
}

// ComputeChecksum computes the SHA256 checksum of the given content.
func ComputeChecksum(content []byte) string {
	hash := sha256.Sum256(content)
	return hex.EncodeToString(hash[:])
}

// ChecksumGenerator generates checksums for bundle files.
type ChecksumGenerator struct {
	result *result.Result
}

// NewChecksumGenerator creates a new checksum generator.
func NewChecksumGenerator(result *result.Result) *ChecksumGenerator {
	return &ChecksumGenerator{
		result: result,
	}
}

// Generate generates a checksums file for all files in the result.
func (g *ChecksumGenerator) Generate(outputDir, title string) (string, error) {
	var content bytes.Buffer
	content.WriteString(fmt.Sprintf("# %s Bundle Checksums (SHA256)\n\n", title))

	for _, file := range g.result.Files {
		// Skip checksums file itself
		if filepath.Base(file) == "checksums.txt" {
			continue
		}

		fileContent, err := os.ReadFile(file)
		if err != nil {
			return "", fmt.Errorf("failed to read file %s for checksum: %w", file, err)
		}

		checksum := ComputeChecksum(fileContent)

		// Get relative path from output directory
		relPath, err := filepath.Rel(outputDir, file)
		if err != nil {
			relPath = filepath.Base(file)
		}

		content.WriteString(fmt.Sprintf("%s  %s\n", checksum, relPath))
	}

	return content.String(), nil
}

// MarshalYAML serializes a value to YAML format.
func MarshalYAML(v interface{}) ([]byte, error) {
	// Import yaml package inline to avoid adding it as a top-level dependency
	// for packages that don't need it
	var buf bytes.Buffer
	encoder := yaml.NewEncoder(&buf)
	encoder.SetIndent(2)
	if err := encoder.Encode(v); err != nil {
		return nil, fmt.Errorf("failed to marshal YAML: %w", err)
	}
	return buf.Bytes(), nil
}

// ValuesHeader contains metadata for values.yaml file headers.
type ValuesHeader struct {
	ComponentName  string
	BundlerVersion string
	RecipeVersion  string
}

// MarshalYAMLWithHeader serializes a value to YAML format with a metadata header.
func MarshalYAMLWithHeader(v interface{}, header ValuesHeader) ([]byte, error) {
	var buf bytes.Buffer

	// Write header comments
	buf.WriteString(fmt.Sprintf("# %s Helm Values\n", header.ComponentName))
	buf.WriteString("# Generated from Cloud Native Stack Recipe\n")
	buf.WriteString(fmt.Sprintf("# Bundler Version: %s\n", header.BundlerVersion))
	buf.WriteString(fmt.Sprintf("# Recipe Version: %s\n", header.RecipeVersion))
	buf.WriteString("\n")

	// Serialize the values
	encoder := yaml.NewEncoder(&buf)
	encoder.SetIndent(2)
	if err := encoder.Encode(v); err != nil {
		return nil, fmt.Errorf("failed to marshal YAML: %w", err)
	}
	return buf.Bytes(), nil
}

// GetConfigValue gets a value from config map with a default fallback.
func GetConfigValue(config map[string]string, key, defaultValue string) string {
	if val, ok := config[key]; ok && val != "" {
		return val
	}

	slog.Debug("config value not found, using default", "key", key, "default", defaultValue)

	return defaultValue
}

// ExtractCustomLabels extracts custom labels from config map with "label_" prefix.
func ExtractCustomLabels(config map[string]string) map[string]string {
	labels := make(map[string]string)
	for k, v := range config {
		if len(k) > 6 && k[:6] == "label_" {
			labels[k[6:]] = v
		}
	}
	return labels
}

// ExtractCustomAnnotations extracts custom annotations from config map with "annotation_" prefix.
func ExtractCustomAnnotations(config map[string]string) map[string]string {
	annotations := make(map[string]string)
	for k, v := range config {
		if len(k) > 11 && k[:11] == "annotation_" {
			annotations[k[11:]] = v
		}
	}
	return annotations
}

// Common string constants for boolean values in Helm templates.
const (
	StrTrue  = "true"
	StrFalse = "false"
)

// BoolToString converts a boolean to "true" or "false" string.
// Use this for Helm values that require string booleans.
func BoolToString(b bool) string {
	if b {
		return StrTrue
	}
	return StrFalse
}

// ParseBoolString parses a string boolean value.
// Returns true if the value is "true" or "1", false otherwise.
func ParseBoolString(s string) bool {
	return s == StrTrue || s == "1"
}
