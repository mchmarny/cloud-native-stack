package serializer

import (
	"testing"
)

func TestParseConfigMapURI(t *testing.T) {
	tests := []struct {
		name          string
		uri           string
		wantNamespace string
		wantName      string
		wantErr       bool
	}{
		{
			name:          "valid URI",
			uri:           "cm://gpu-operator/eidos-snapshot",
			wantNamespace: "gpu-operator",
			wantName:      "eidos-snapshot",
			wantErr:       false,
		},
		{
			name:          "valid URI with spaces",
			uri:           "cm://gpu-operator / eidos-snapshot ",
			wantNamespace: "gpu-operator",
			wantName:      "eidos-snapshot",
			wantErr:       false,
		},
		{
			name:          "valid URI with default namespace",
			uri:           "cm://default/snapshot",
			wantNamespace: "default",
			wantName:      "snapshot",
			wantErr:       false,
		},
		{
			name:    "missing scheme",
			uri:     "gpu-operator/eidos-snapshot",
			wantErr: true,
		},
		{
			name:    "wrong scheme",
			uri:     "http://gpu-operator/eidos-snapshot",
			wantErr: true,
		},
		{
			name:    "missing name",
			uri:     "cm://gpu-operator/",
			wantErr: true,
		},
		{
			name:    "missing namespace",
			uri:     "cm:///eidos-snapshot",
			wantErr: true,
		},
		{
			name:    "missing separator",
			uri:     "cm://gpu-operator",
			wantErr: true,
		},
		{
			name:    "empty URI",
			uri:     "",
			wantErr: true,
		},
		{
			name:    "only scheme",
			uri:     "cm://",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			namespace, name, err := parseConfigMapURI(tt.uri)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseConfigMapURI() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if namespace != tt.wantNamespace {
					t.Errorf("parseConfigMapURI() namespace = %v, want %v", namespace, tt.wantNamespace)
				}
				if name != tt.wantName {
					t.Errorf("parseConfigMapURI() name = %v, want %v", name, tt.wantName)
				}
			}
		})
	}
}

func TestNewConfigMapWriter(t *testing.T) {
	tests := []struct {
		name       string
		namespace  string
		cmName     string
		format     Format
		wantFormat Format
	}{
		{
			name:       "valid JSON format",
			namespace:  "default",
			cmName:     "test",
			format:     FormatJSON,
			wantFormat: FormatJSON,
		},
		{
			name:       "valid YAML format",
			namespace:  "gpu-operator",
			cmName:     "snapshot",
			format:     FormatYAML,
			wantFormat: FormatYAML,
		},
		{
			name:       "unknown format defaults to JSON",
			namespace:  "default",
			cmName:     "test",
			format:     Format("unknown"),
			wantFormat: FormatJSON,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			writer := NewConfigMapWriter(tt.namespace, tt.cmName, tt.format)
			if writer.namespace != tt.namespace {
				t.Errorf("NewConfigMapWriter() namespace = %v, want %v", writer.namespace, tt.namespace)
			}
			if writer.name != tt.cmName {
				t.Errorf("NewConfigMapWriter() name = %v, want %v", writer.name, tt.cmName)
			}
			if writer.format != tt.wantFormat {
				t.Errorf("NewConfigMapWriter() format = %v, want %v", writer.format, tt.wantFormat)
			}
		})
	}
}
