package dradriver

import (
	_ "embed"
)

//go:embed templates/README.md.tmpl
var readmeTemplate string

// GetTemplate returns the named template content for README and manifest generation.
func GetTemplate(name string) (string, bool) {
	templates := map[string]string{
		"README.md": readmeTemplate,
	}

	tmpl, ok := templates[name]
	return tmpl, ok
}
