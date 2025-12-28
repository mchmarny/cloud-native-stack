package config

import "fmt"

// Config provides configuration options for bundlers.
type Config struct {
	// OutputFormat specifies the format for generated files.
	// Supported: "yaml", "json", "helm"
	OutputFormat string `json:"output_format" yaml:"output_format"`

	// Compression enables tar.gz compression of the bundle.
	Compression bool `json:"compression" yaml:"compression"`

	// IncludeScripts includes installation and setup scripts.
	IncludeScripts bool `json:"include_scripts" yaml:"include_scripts"`

	// IncludeReadme includes README documentation.
	IncludeReadme bool `json:"include_readme" yaml:"include_readme"`

	// IncludeChecksums includes checksum file for verification.
	IncludeChecksums bool `json:"include_checksums" yaml:"include_checksums"`

	// HelmChartVersion specifies the Helm chart version to use.
	HelmChartVersion string `json:"helm_chart_version,omitempty" yaml:"helm_chart_version,omitempty"`

	// HelmRepository specifies the Helm repository URL.
	HelmRepository string `json:"helm_repository,omitempty" yaml:"helm_repository,omitempty"`

	// CustomLabels adds custom labels to generated resources.
	CustomLabels map[string]string `json:"custom_labels,omitempty" yaml:"custom_labels,omitempty"`

	// CustomAnnotations adds custom annotations to generated resources.
	CustomAnnotations map[string]string `json:"custom_annotations,omitempty" yaml:"custom_annotations,omitempty"`

	// Namespace specifies the Kubernetes namespace for resources.
	Namespace string `json:"namespace,omitempty" yaml:"namespace,omitempty"`

	// Verbose enables detailed output during bundle generation.
	Verbose bool `json:"verbose" yaml:"verbose"`
}

// Validate checks if the Config has valid settings.
func (c *Config) Validate() error {
	validFormats := map[string]bool{"yaml": true, "json": true, "helm": true}
	if !validFormats[c.OutputFormat] {
		return fmt.Errorf("invalid output format: %s (must be yaml, json, or helm)",
			c.OutputFormat)
	}

	if c.Namespace == "" {
		return fmt.Errorf("namespace cannot be empty")
	}

	return nil
}

// NewConfig returns a Config with default values.
func NewConfig() *Config {
	return &Config{
		OutputFormat:      "yaml",
		Compression:       false,
		IncludeScripts:    true,
		IncludeReadme:     true,
		IncludeChecksums:  true,
		CustomLabels:      make(map[string]string),
		CustomAnnotations: make(map[string]string),
		Namespace:         "default",
		Verbose:           false,
	}
}

// Merge merges another config into this one, with the other config taking precedence.
func (c *Config) Merge(other *Config) {
	if other == nil {
		return
	}

	if other.OutputFormat != "" {
		c.OutputFormat = other.OutputFormat
	}
	c.Compression = other.Compression
	c.IncludeScripts = other.IncludeScripts
	c.IncludeReadme = other.IncludeReadme
	c.IncludeChecksums = other.IncludeChecksums

	if other.HelmChartVersion != "" {
		c.HelmChartVersion = other.HelmChartVersion
	}
	if other.HelmRepository != "" {
		c.HelmRepository = other.HelmRepository
	}
	if other.Namespace != "" {
		c.Namespace = other.Namespace
	}

	// Merge labels and annotations
	for k, v := range other.CustomLabels {
		c.CustomLabels[k] = v
	}
	for k, v := range other.CustomAnnotations {
		c.CustomAnnotations[k] = v
	}

	c.Verbose = other.Verbose
}

// Clone creates a deep copy of the config.
func (c *Config) Clone() *Config {
	clone := *c
	clone.CustomLabels = make(map[string]string, len(c.CustomLabels))
	clone.CustomAnnotations = make(map[string]string, len(c.CustomAnnotations))

	for k, v := range c.CustomLabels {
		clone.CustomLabels[k] = v
	}
	for k, v := range c.CustomAnnotations {
		clone.CustomAnnotations[k] = v
	}

	return &clone
}
