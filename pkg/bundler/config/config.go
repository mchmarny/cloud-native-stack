package config

import (
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
)

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

	// verbose enables detailed output during bundle generation.
	verbose bool

	// version specifies the bundler version.
	version string

	// valueOverrides contains user-specified value overrides per bundler.
	// Map structure: bundler_name -> (path -> value)
	valueOverrides map[string]map[string]string

	// systemNodeSelector contains node selector labels for system components.
	systemNodeSelector map[string]string

	// systemNodeTolerations contains tolerations for system components.
	systemNodeTolerations []corev1.Toleration

	// acceleratedNodeSelector contains node selector labels for accelerated/GPU nodes.
	acceleratedNodeSelector map[string]string

	// acceleratedNodeTolerations contains tolerations for accelerated/GPU nodes.
	acceleratedNodeTolerations []corev1.Toleration
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

// SystemNodeSelector returns a copy of the system node selector map.
func (c *Config) SystemNodeSelector() map[string]string {
	if c.systemNodeSelector == nil {
		return nil
	}
	result := make(map[string]string, len(c.systemNodeSelector))
	for k, v := range c.systemNodeSelector {
		result[k] = v
	}
	return result
}

// SystemNodeTolerations returns a copy of the system node tolerations.
func (c *Config) SystemNodeTolerations() []corev1.Toleration {
	if c.systemNodeTolerations == nil {
		return nil
	}
	result := make([]corev1.Toleration, len(c.systemNodeTolerations))
	copy(result, c.systemNodeTolerations)
	return result
}

// AcceleratedNodeSelector returns a copy of the accelerated node selector map.
func (c *Config) AcceleratedNodeSelector() map[string]string {
	if c.acceleratedNodeSelector == nil {
		return nil
	}
	result := make(map[string]string, len(c.acceleratedNodeSelector))
	for k, v := range c.acceleratedNodeSelector {
		result[k] = v
	}
	return result
}

// AcceleratedNodeTolerations returns a copy of the accelerated node tolerations.
func (c *Config) AcceleratedNodeTolerations() []corev1.Toleration {
	if c.acceleratedNodeTolerations == nil {
		return nil
	}
	result := make([]corev1.Toleration, len(c.acceleratedNodeTolerations))
	copy(result, c.acceleratedNodeTolerations)
	return result
}

// Validate checks if the Config has valid settings.
func (c *Config) Validate() error {
	validFormats := map[string]bool{"yaml": true, "json": true, "helm": true}
	if !validFormats[c.outputFormat] {
		return fmt.Errorf("invalid output format: %s (must be yaml, json, or helm)",
			c.outputFormat)
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

// WithSystemNodeSelector sets the node selector for system components.
func WithSystemNodeSelector(selector map[string]string) Option {
	return func(c *Config) {
		if selector == nil {
			return
		}
		c.systemNodeSelector = make(map[string]string, len(selector))
		for k, v := range selector {
			c.systemNodeSelector[k] = v
		}
	}
}

// WithSystemNodeTolerations sets the tolerations for system components.
func WithSystemNodeTolerations(tolerations []corev1.Toleration) Option {
	return func(c *Config) {
		if tolerations == nil {
			return
		}
		c.systemNodeTolerations = make([]corev1.Toleration, len(tolerations))
		copy(c.systemNodeTolerations, tolerations)
	}
}

// WithAcceleratedNodeSelector sets the node selector for accelerated/GPU nodes.
func WithAcceleratedNodeSelector(selector map[string]string) Option {
	return func(c *Config) {
		if selector == nil {
			return
		}
		c.acceleratedNodeSelector = make(map[string]string, len(selector))
		for k, v := range selector {
			c.acceleratedNodeSelector[k] = v
		}
	}
}

// WithAcceleratedNodeTolerations sets the tolerations for accelerated/GPU nodes.
func WithAcceleratedNodeTolerations(tolerations []corev1.Toleration) Option {
	return func(c *Config) {
		if tolerations == nil {
			return
		}
		c.acceleratedNodeTolerations = make([]corev1.Toleration, len(tolerations))
		copy(c.acceleratedNodeTolerations, tolerations)
	}
}

// NewConfig returns a Config with default values.
func NewConfig(options ...Option) *Config {
	c := &Config{
		compression:      false,
		includeChecksums: true,
		includeReadme:    true,
		includeScripts:   true,
		outputFormat:     "yaml",
		valueOverrides:   make(map[string]map[string]string),
		verbose:          false,
		version:          "dev",
	}
	for _, opt := range options {
		opt(c)
	}
	return c
}

// ParseValueOverrides parses value override strings in format "bundler:path.to.field=value".
// Returns a map of bundler -> (path -> value).
// This function is used by both CLI and API handlers to parse --set flags and query parameters.
func ParseValueOverrides(overrides []string) (map[string]map[string]string, error) {
	result := make(map[string]map[string]string)

	for _, override := range overrides {
		// Split on first ':' to get bundler and path=value
		parts := strings.SplitN(override, ":", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid format '%s': expected 'bundler:path=value'", override)
		}

		bundlerName := parts[0]
		pathValue := parts[1]

		// Split on first '=' to get path and value
		kvParts := strings.SplitN(pathValue, "=", 2)
		if len(kvParts) != 2 {
			return nil, fmt.Errorf("invalid format '%s': expected 'bundler:path=value'", override)
		}

		path := kvParts[0]
		value := kvParts[1]

		if path == "" || value == "" {
			return nil, fmt.Errorf("invalid format '%s': path and value cannot be empty", override)
		}

		// Initialize bundler map if needed
		if result[bundlerName] == nil {
			result[bundlerName] = make(map[string]string)
		}

		result[bundlerName][path] = value
	}

	return result, nil
}
