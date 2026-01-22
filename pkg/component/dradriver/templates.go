package dradriver

import (
	_ "embed"
)

//go:embed templates/install.sh.tmpl
var installScriptTemplate string

//go:embed templates/uninstall.sh.tmpl
var uninstallScriptTemplate string

//go:embed templates/README.md.tmpl
var readmeTemplate string

// GetTemplate returns the named template content.
func GetTemplate(name string) (string, bool) {
	templates := map[string]string{
		"install.sh":   installScriptTemplate,
		"uninstall.sh": uninstallScriptTemplate,
		"README.md":    readmeTemplate,
	}

	tmpl, ok := templates[name]
	return tmpl, ok
}
