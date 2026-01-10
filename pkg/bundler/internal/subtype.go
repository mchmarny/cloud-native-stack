package internal

import "log/slog"

// ValueWithContext holds a configuration value with its context/explanation.
// This is used to track both the value and the reason/source for that value
// from the recipe measurements.
type ValueWithContext struct {
	Value   interface{}
	Context string // Human-readable explanation from recipe
}

// GetConfigValue gets a value from config map with a default fallback.
func GetConfigValue(config map[string]string, key, defaultValue string) string {
	if val, ok := config[key]; ok && val != "" {
		return val
	}

	slog.Debug("config value not found, using default", "key", key, "default", defaultValue)

	return defaultValue
}

// GetSubtypeContext extracts the general context from subtype context map.
func GetSubtypeContext(contextMap map[string]string) string {
	if desc, ok := contextMap["description"]; ok && desc != "" {
		return desc
	}
	if reason, ok := contextMap["reason"]; ok && reason != "" {
		return reason
	}
	return ""
}

// GetFieldContext gets the context for a specific field, falling back to subtype context.
func GetFieldContext(contextMap map[string]string, fieldName, subtypeContext string) string {
	// Try field-specific context first (e.g., "gpu-operator-context")
	if ctx, ok := contextMap[fieldName+"-context"]; ok && ctx != "" {
		return ctx
	}
	if ctx, ok := contextMap[fieldName]; ok && ctx != "" {
		return ctx
	}
	// Fall back to subtype-level context
	return subtypeContext
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

// ApplyValueOverrides applies user-specified value overrides to a ValueWithContext field.
// The path uses dot notation (e.g., "gds.enabled") which is converted to field paths.
// This allows users to override recipe values via CLI flags.
func ApplyValueOverrides(fieldPath string, current ValueWithContext, overrides map[string]string) ValueWithContext {
	if overrides == nil {
		return current
	}

	// Check if there's an override for this exact path
	if overrideValue, ok := overrides[fieldPath]; ok {
		return ValueWithContext{
			Value:   overrideValue,
			Context: "User override via --set flag",
		}
	}

	return current
}
