package gpuoperator

import (
	_ "embed"
)

//go:embed templates/README.md.tmpl
var readmeTemplate string

//go:embed templates/kernel-module-params.yaml.tmpl
var kernelModuleParamsTemplate string

//go:embed templates/dcgm-exporter.yaml.tmpl
var dcgmExporterTemplate string

// GetTemplate returns the named template content for README and manifest generation.
func GetTemplate(name string) (string, bool) {
	templates := map[string]string{
		"README.md":            readmeTemplate,
		"kernel-module-params": kernelModuleParamsTemplate,
		"dcgm-exporter":        dcgmExporterTemplate,
	}

	tmpl, ok := templates[name]
	return tmpl, ok
}
