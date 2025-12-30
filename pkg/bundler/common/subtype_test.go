package common

import (
	"testing"
)

func TestGetConfigValue(t *testing.T) {
	tests := []struct {
		name         string
		config       map[string]string
		key          string
		defaultValue string
		want         string
	}{
		{
			name:         "key exists",
			config:       map[string]string{"key": "value"},
			key:          "key",
			defaultValue: "default",
			want:         "value",
		},
		{
			name:         "key missing",
			config:       map[string]string{},
			key:          "key",
			defaultValue: "default",
			want:         "default",
		},
		{
			name:         "empty value uses default",
			config:       map[string]string{"key": ""},
			key:          "key",
			defaultValue: "default",
			want:         "default",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetConfigValue(tt.config, tt.key, tt.defaultValue)
			if got != tt.want {
				t.Errorf("GetConfigValue() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetSubtypeContext(t *testing.T) {
	tests := []struct {
		name       string
		contextMap map[string]string
		want       string
	}{
		{
			name:       "has description",
			contextMap: map[string]string{"description": "test desc"},
			want:       "test desc",
		},
		{
			name:       "has reason",
			contextMap: map[string]string{"reason": "test reason"},
			want:       "test reason",
		},
		{
			name:       "description takes precedence",
			contextMap: map[string]string{"description": "desc", "reason": "reason"},
			want:       "desc",
		},
		{
			name:       "empty map",
			contextMap: map[string]string{},
			want:       "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetSubtypeContext(tt.contextMap)
			if got != tt.want {
				t.Errorf("GetSubtypeContext() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetFieldContext(t *testing.T) {
	tests := []struct {
		name           string
		contextMap     map[string]string
		fieldName      string
		subtypeContext string
		want           string
	}{
		{
			name:           "has field-specific context",
			contextMap:     map[string]string{"gpu-operator-context": "gpu context"},
			fieldName:      "gpu-operator",
			subtypeContext: "subtype context",
			want:           "gpu context",
		},
		{
			name:           "has field without -context suffix",
			contextMap:     map[string]string{"gpu-operator": "gpu value"},
			fieldName:      "gpu-operator",
			subtypeContext: "subtype context",
			want:           "gpu value",
		},
		{
			name:           "falls back to subtype context",
			contextMap:     map[string]string{},
			fieldName:      "gpu-operator",
			subtypeContext: "subtype context",
			want:           "subtype context",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetFieldContext(tt.contextMap, tt.fieldName, tt.subtypeContext)
			if got != tt.want {
				t.Errorf("GetFieldContext() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestExtractCustomLabels(t *testing.T) {
	tests := []struct {
		name   string
		config map[string]string
		want   map[string]string
	}{
		{
			name: "extracts labels",
			config: map[string]string{
				"label_env":  "prod",
				"label_team": "platform",
				"other_key":  "value",
			},
			want: map[string]string{
				"env":  "prod",
				"team": "platform",
			},
		},
		{
			name:   "empty config",
			config: map[string]string{},
			want:   map[string]string{},
		},
		{
			name: "no labels",
			config: map[string]string{
				"key": "value",
			},
			want: map[string]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractCustomLabels(tt.config)
			if len(got) != len(tt.want) {
				t.Errorf("ExtractCustomLabels() len = %v, want %v", len(got), len(tt.want))
			}
			for k, v := range tt.want {
				if got[k] != v {
					t.Errorf("ExtractCustomLabels()[%v] = %v, want %v", k, got[k], v)
				}
			}
		})
	}
}

func TestExtractCustomAnnotations(t *testing.T) {
	tests := []struct {
		name   string
		config map[string]string
		want   map[string]string
	}{
		{
			name: "extracts annotations",
			config: map[string]string{
				"annotation_key1": "value1",
				"annotation_key2": "value2",
				"other_key":       "value",
			},
			want: map[string]string{
				"key1": "value1",
				"key2": "value2",
			},
		},
		{
			name:   "empty config",
			config: map[string]string{},
			want:   map[string]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractCustomAnnotations(tt.config)
			if len(got) != len(tt.want) {
				t.Errorf("ExtractCustomAnnotations() len = %v, want %v", len(got), len(tt.want))
			}
			for k, v := range tt.want {
				if got[k] != v {
					t.Errorf("ExtractCustomAnnotations()[%v] = %v, want %v", k, got[k], v)
				}
			}
		})
	}
}
