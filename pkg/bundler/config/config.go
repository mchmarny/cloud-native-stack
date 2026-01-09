package config

import "fmt"

// Config provides immutable configuration options for bundlers.
// All fields are read-only after creation to prevent accidental modifications.
// Use Clone() to create a modified copy or Merge() to combine configurations.
type Config struct {
	// outputFormat specifies the format for generated files.
	// Supported: "yaml", "json", "helm"
	outputFormat string

	// compression enables tar.gz compression of the bundle.
	compression bool

	// includeScripts includes installation and setup scripts.
	includeScripts bool

	// includeReadme includes README documentation.
	includeReadme bool

	// includeChecksums includes checksum file for verification.
	includeChecksums bool

	// helmChartVersion specifies the Helm chart version to use.
	helmChartVersion string

	// helmRepository specifies the Helm repository URL.
	helmRepository string

	// customLabels adds custom labels to generated resources.
	customLabels map[string]string

	// customAnnotations adds custom annotations to generated resources.
	customAnnotations map[string]string

	// namespace specifies the Kubernetes namespace for resources.
	namespace string

	// verbose enables detailed output during bundle generation.
	verbose bool

	// version specifies the bundler version.
	version string

	// valueOverrides contains user-specified value overrides per bundler.
	// Map structure: bundler_name -> (path -> value)
	valueOverrides map[string]map[string]string
}

// Getter methods for read-only access

// OutputFormat returns the output format setting.
func (c *Config) OutputFormat() string {
	return c.outputFormat
}

// Compression returns the compression setting.
func (c *Config) Compression() bool {
	return c.compression
}

// IncludeScripts returns the include scripts setting.
func (c *Config) IncludeScripts() bool {
	return c.includeScripts
}

// IncludeReadme returns the include readme setting.
func (c *Config) IncludeReadme() bool {
	return c.includeReadme
}

// IncludeChecksums returns the include checksums setting.
func (c *Config) IncludeChecksums() bool {
	return c.includeChecksums
}

// HelmChartVersion returns the Helm chart version setting.
func (c *Config) HelmChartVersion() string {
	return c.helmChartVersion
}

// HelmRepository returns the Helm repository URL setting.
func (c *Config) HelmRepository() string {
	return c.helmRepository
}

// CustomLabels returns a copy of the custom labels to prevent modification.
func (c *Config) CustomLabels() map[string]string {
	labels := make(map[string]string, len(c.customLabels))
	for k, v := range c.customLabels {
		labels[k] = v
	}
	return labels
}

// CustomAnnotations returns a copy of the custom annotations to prevent modification.
func (c *Config) CustomAnnotations() map[string]string {
	annotations := make(map[string]string, len(c.customAnnotations))
	for k, v := range c.customAnnotations {
		annotations[k] = v
	}
	return annotations
}

// Namespace returns the namespace setting.
func (c *Config) Namespace() string {
	return c.namespace
}

// Verbose returns the verbose setting.
func (c *Config) Verbose() bool {
	return c.verbose
}

// Version returns the bundler version.
func (c *Config) Version() string {
	return c.version
}

// ValueOverrides returns a deep copy of the value overrides to prevent modification.
func (c *Config) ValueOverrides() map[string]map[string]string {
	if c.valueOverrides == nil {
		return nil
	}
	overrides := make(map[string]map[string]string, len(c.valueOverrides))
	for bundler, paths := range c.valueOverrides {
		overrides[bundler] = make(map[string]string, len(paths))
		for path, value := range paths {
			overrides[bundler][path] = value
		}
	}
	return overrides
}

// Validate checks if the Config has valid settings.
func (c *Config) Validate() error {
	validFormats := map[string]bool{"yaml": true, "json": true, "helm": true}
	if !validFormats[c.outputFormat] {
		return fmt.Errorf("invalid output format: %s (must be yaml, json, or helm)",
			c.outputFormat)
	}

	if c.namespace == "" {
		return fmt.Errorf("namespace cannot be empty")
	}

	return nil
}

type Option func(*Config)

// WithOutputFormat sets the output format for the bundler (yaml, json, helm).
func WithOutputFormat(format string) Option {
	return func(c *Config) {
		c.outputFormat = format
	}
}

// WithCompression sets whether compression is enabled for the bundler.
func WithCompression(enabled bool) Option {
	return func(c *Config) {
		c.compression = enabled
	}
}

// WithIncludeScripts sets whether installation and uninstallation scripts should be included in the bundle.
func WithIncludeScripts(enabled bool) Option {
	return func(c *Config) {
		c.includeScripts = enabled
	}
}

// WithIncludeReadme sets whether a README should be included in the bundle.
func WithIncludeReadme(enabled bool) Option {
	return func(c *Config) {
		c.includeReadme = enabled
	}
}

// WithIncludeChecksums sets whether a checksums file should be included in the bundle.
func WithIncludeChecksums(enabled bool) Option {
	return func(c *Config) {
		c.includeChecksums = enabled
	}
}

// WithHelmChartVersion sets the Helm chart version for the bundler.
func WithHelmChartVersion(version string) Option {
	return func(c *Config) {
		c.helmChartVersion = version
	}
}

// WithHelmRepository sets the Helm repository URL for the bundler.
func WithHelmRepository(url string) Option {
	return func(c *Config) {
		c.helmRepository = url
	}
}

// WithCustomLabels sets custom labels for the bundler.
func WithCustomLabels(labels map[string]string) Option {
	return func(c *Config) {
		for k, v := range labels {
			c.customLabels[k] = v
		}
	}
}

// WithCustomAnnotations sets custom annotations for the bundler.
func WithCustomAnnotations(annotations map[string]string) Option {
	return func(c *Config) {
		for k, v := range annotations {
			c.customAnnotations[k] = v
		}
	}
}

// WithNamespace sets the namespace for the bundler.
func WithNamespace(namespace string) Option {
	return func(c *Config) {
		c.namespace = namespace
	}
}

// WithVerbose sets whether verbose logging is enabled for the bundler.
func WithVerbose(enabled bool) Option {
	return func(c *Config) {
		c.verbose = enabled
	}
}

// WithVersion sets the version for the bundler.
func WithVersion(version string) Option {
	return func(c *Config) {
		c.version = version
	}
}

// WithValueOverrides sets value overrides for the bundler.
func WithValueOverrides(overrides map[string]map[string]string) Option {
	return func(c *Config) {
		if overrides == nil {
			return
		}
		// Deep copy to prevent external modifications
		for bundler, paths := range overrides {
			if c.valueOverrides[bundler] == nil {
				c.valueOverrides[bundler] = make(map[string]string)
			}
			for path, value := range paths {
				c.valueOverrides[bundler][path] = value
			}
		}
	}
}

// NewConfig returns a Config with default values.
func NewConfig(options ...Option) *Config {
	c := &Config{
		compression:       false,
		customAnnotations: make(map[string]string),
		customLabels:      make(map[string]string),
		includeChecksums:  true,
		includeReadme:     true,
		includeScripts:    true,
		namespace:         "default",
		outputFormat:      "yaml",
		valueOverrides:    make(map[string]map[string]string),
		verbose:           false,
		version:           "dev",
	}
	for _, opt := range options {
		opt(c)
	}
	return c
}
