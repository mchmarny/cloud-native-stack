package gpuoperator

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
	"time"

	"github.com/NVIDIA/cloud-native-stack/pkg/bundler/bundle"
	"github.com/NVIDIA/cloud-native-stack/pkg/bundler/config"
	"github.com/NVIDIA/cloud-native-stack/pkg/errors"
	"github.com/NVIDIA/cloud-native-stack/pkg/measurement"
	"github.com/NVIDIA/cloud-native-stack/pkg/recipe"
)

// Bundler creates GPU Operator application bundles based on recipes.
type Bundler struct {
	config *config.Config
}

// NewBundler creates a new GPU Operator bundler instance.
func NewBundler(conf *config.Config) *Bundler {
	if conf == nil {
		conf = config.NewConfig()
	}

	return &Bundler{
		config: conf,
	}
}

// Make generates the GPU Operator bundle based on the provided recipe.
func (b *Bundler) Make(ctx context.Context, recipe *recipe.Recipe, dir string) (*bundle.Result, error) {
	// Check for required measurements
	if err := recipe.ValidateMeasurementExists(measurement.TypeK8s); err != nil {
		return nil, fmt.Errorf("measurements are required for GPU Operator bundling: %w", err)
	}

	// Check for GPU measurements (optional but recommended)
	if err := recipe.ValidateMeasurementExists(measurement.TypeGPU); err != nil {
		slog.Warn("GPU measurements not found in recipe", "warning", err)
	}

	start := time.Now()
	result := bundle.NewResult(bundle.BundleTypeGpuOperator)

	slog.Debug("generating GPU Operator bundle",
		"output_dir", dir,
		"namespace", b.config.Namespace,
	)

	// Create bundle directory structure
	bundleDir := filepath.Join(dir, "gpu-operator")
	if err := os.MkdirAll(bundleDir, 0755); err != nil {
		return result, errors.Wrap(errors.ErrCodeInternal,
			"failed to create bundle directory", err)
	}

	// Create subdirectories
	scriptsDir := filepath.Join(bundleDir, "scripts")
	manifestsDir := filepath.Join(bundleDir, "manifests")

	for _, subdir := range []string{scriptsDir, manifestsDir} {
		if err := os.MkdirAll(subdir, 0755); err != nil {
			return result, errors.Wrap(errors.ErrCodeInternal,
				"failed to create subdirectory", err)
		}
	}

	// Prepare configuration map
	configMap := b.buildConfigMap()

	// Generate Helm values
	if err := b.generateHelmValues(ctx, recipe, bundleDir, configMap, result); err != nil {
		return result, err
	}

	// Generate ClusterPolicy manifest
	if err := b.generateClusterPolicy(ctx, recipe, manifestsDir, configMap, result); err != nil {
		return result, err
	}

	// Generate installation scripts
	if b.config.IncludeScripts {
		if err := b.generateScripts(ctx, recipe, scriptsDir, configMap, result); err != nil {
			return result, err
		}
	}

	// Generate README
	if b.config.IncludeReadme {
		if err := b.generateReadme(ctx, recipe, bundleDir, configMap, result); err != nil {
			return result, err
		}
	}

	// Generate checksums file
	if b.config.IncludeChecksums {
		if err := b.generateChecksums(bundleDir, result); err != nil {
			return result, err
		}
	}

	result.Duration = time.Since(start)

	// Mark the result as successful
	result.MarkSuccess()

	slog.Info("GPU Operator bundle generated",
		"files", len(result.Files),
		"size_bytes", result.Size,
		"duration", result.Duration.Round(time.Millisecond),
	)

	return result, nil
}

// buildConfigMap creates a configuration map from bundler config.
func (b *Bundler) buildConfigMap() map[string]string {
	config := make(map[string]string)
	config["namespace"] = b.config.Namespace
	config["helm_repository"] = b.config.HelmRepository
	config["helm_chart_version"] = b.config.HelmChartVersion

	// Add custom labels as config
	for k, v := range b.config.CustomLabels {
		config["label_"+k] = v
	}

	// Add custom annotations as config
	for k, v := range b.config.CustomAnnotations {
		config["annotation_"+k] = v
	}

	return config
}

// generateHelmValues generates Helm values file.
func (b *Bundler) generateHelmValues(ctx context.Context, recipe *recipe.Recipe,
	bundleDir string, config map[string]string, result *bundle.Result) error {

	// Check context cancellation
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	helmValues := GenerateHelmValues(recipe, config)

	if errValidate := helmValues.Validate(); errValidate != nil {
		return errors.Wrap(errors.ErrCodeInvalidRequest, "invalid helm values", errValidate)
	}

	content, err := b.renderTemplate("values.yaml", helmValues.ToMap())
	if err != nil {
		return errors.Wrap(errors.ErrCodeInternal, "failed to render values template", err)
	}

	filePath := filepath.Join(bundleDir, "values.yaml")
	if err := b.writeFile(filePath, []byte(content), result); err != nil {
		return err
	}

	return nil
}

// generateClusterPolicy generates ClusterPolicy manifest.
func (b *Bundler) generateClusterPolicy(ctx context.Context, recipe *recipe.Recipe,
	dir string, config map[string]string, result *bundle.Result) error {

	// Check context cancellation
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	manifestData := GenerateManifestData(recipe, config)

	content, err := b.renderTemplate("clusterpolicy", manifestData.ToMap())
	if err != nil {
		return errors.Wrap(errors.ErrCodeInternal, "failed to render clusterpolicy template", err)
	}

	filePath := filepath.Join(dir, "clusterpolicy.yaml")
	if err := b.writeFile(filePath, []byte(content), result); err != nil {
		return err
	}

	return nil
}

// generateScripts generates installation and uninstallation scripts.
func (b *Bundler) generateScripts(ctx context.Context, recipe *recipe.Recipe,
	dir string, config map[string]string, result *bundle.Result) error {

	// Check context cancellation
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	scriptData := GenerateScriptData(recipe, config)

	// Generate install script
	installContent, err := b.renderTemplate("install.sh", scriptData.ToMap())
	if err != nil {
		return errors.Wrap(errors.ErrCodeInternal, "failed to render install script", err)
	}

	installPath := filepath.Join(dir, "install.sh")
	if errWrite := b.writeFile(installPath, []byte(installContent), result); errWrite != nil {
		return errWrite
	}
	// Make executable
	if errChmod := os.Chmod(installPath, 0755); errChmod != nil {
		result.AddError(fmt.Errorf("failed to make install.sh executable: %w", errChmod))
	}

	// Generate uninstall script
	uninstallContent, err := b.renderTemplate("uninstall.sh", scriptData.ToMap())
	if err != nil {
		return errors.Wrap(errors.ErrCodeInternal, "failed to render uninstall script", err)
	}

	uninstallPath := filepath.Join(dir, "uninstall.sh")
	if err := b.writeFile(uninstallPath, []byte(uninstallContent), result); err != nil {
		return err
	}
	// Make executable
	if err := os.Chmod(uninstallPath, 0755); err != nil {
		result.AddError(fmt.Errorf("failed to make uninstall.sh executable: %w", err))
	}

	return nil
}

// generateReadme generates README documentation.
func (b *Bundler) generateReadme(ctx context.Context, recipe *recipe.Recipe,
	dir string, config map[string]string, result *bundle.Result) error {

	// Check context cancellation
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	scriptData := GenerateScriptData(recipe, config)

	content, err := b.renderTemplate("README.md", scriptData.ToMap())
	if err != nil {
		return errors.Wrap(errors.ErrCodeInternal, "failed to render README template", err)
	}

	filePath := filepath.Join(dir, "README.md")
	if err := b.writeFile(filePath, []byte(content), result); err != nil {
		return err
	}

	return nil
}

// generateChecksums generates a checksums file for bundle verification.
func (b *Bundler) generateChecksums(dir string, result *bundle.Result) error {
	checksums := &bytes.Buffer{}
	checksums.WriteString("# GPU Operator Bundle Checksums (SHA256)\n")
	fmt.Fprintf(checksums, "# Generated: %s\n\n", time.Now().UTC().Format(time.RFC3339))

	for _, file := range result.Files {
		content, err := os.ReadFile(file)
		if err != nil {
			return errors.Wrap(errors.ErrCodeInternal,
				fmt.Sprintf("failed to read file %s for checksum", file), err)
		}

		hash := sha256.Sum256(content)
		checksum := hex.EncodeToString(hash[:])
		relPath, _ := filepath.Rel(filepath.Dir(dir), file)
		fmt.Fprintf(checksums, "%s  %s\n", checksum, relPath)
	}

	filePath := filepath.Join(dir, "checksums.txt")
	if err := b.writeFile(filePath, checksums.Bytes(), result); err != nil {
		return err
	}

	return nil
}

// renderTemplate renders a template with the given data.
func (b *Bundler) renderTemplate(name string, data map[string]interface{}) (string, error) {
	tmplContent, ok := GetTemplate(name)
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

// writeFile writes content to a file and updates the result.
func (b *Bundler) writeFile(path string, content []byte, result *bundle.Result) error {
	if err := os.WriteFile(path, content, 0600); err != nil {
		return errors.Wrap(errors.ErrCodeInternal,
			fmt.Sprintf("failed to write file %s", path), err)
	}

	result.AddFile(path, int64(len(content)))

	slog.Debug("file written",
		"path", path,
		"size_bytes", len(content),
	)

	return nil
}
