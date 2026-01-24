package skyhook

import (
	_ "embed"
)

//go:embed templates/README.md.tmpl
var readmeTemplate string

// Customization templates - each customization is a separate embedded file
//
//go:embed templates/customizations/ubuntu.yaml.tmpl
var ubuntuCustomizationTemplate string

// customizationTemplates maps customization names to their template content.
// Add new customizations here as they are created.
var customizationTemplates = map[string]string{
	"ubuntu": ubuntuCustomizationTemplate,
}

// GetTemplate returns the named template content.
func GetTemplate(name string) (string, bool) {
	templates := map[string]string{
		"README.md": readmeTemplate,
	}
	tmpl, ok := templates[name]
	return tmpl, ok
}

// GetCustomizationTemplate returns the template for a specific customization.
// Returns the template content and true if found, empty string and false otherwise.
func GetCustomizationTemplate(name string) (string, bool) {
	tmpl, ok := customizationTemplates[name]
	return tmpl, ok
}

// ListCustomizations returns all available customization names.
func ListCustomizations() []string {
	names := make([]string, 0, len(customizationTemplates))
	for name := range customizationTemplates {
		names = append(names, name)
	}
	return names
}
