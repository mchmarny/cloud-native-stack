package serializers

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"reflect"
	"sort"
	"strings"
	"text/tabwriter"

	"gopkg.in/yaml.v3"
)

// Format represents the output format type
type Format string

const (
	// FormatJSON outputs data in JSON format
	FormatJSON Format = "json"
	// FormatYAML outputs data in YAML format
	FormatYAML Format = "yaml"
	// FormatTable outputs data in table format
	FormatTable Format = "table"
)

func (f Format) IsUnknown() bool {
	switch f {
	case FormatJSON, FormatYAML, FormatTable:
		return false
	default:
		return true
	}
}

// Writer handles serialization of configuration data to various formats.
type Writer struct {
	format Format
	output io.Writer
}

// NewWriter creates a new Writer with the specified format and output destination.
// If output is nil, os.Stdout will be used.
func NewWriter(format Format, output io.Writer) *Writer {
	if output == nil {
		output = os.Stdout
	}
	return &Writer{
		format: format,
		output: output,
	}
}

// NewFileWriterOrPanic creates a new Writer that outputs to the specified file path in the given format.
// It panics if the file cannot be created.
func NewFileWriterOrStdout(format Format, path string) *Writer {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return NewStdoutWriter(format)
	}

	file, err := os.Create(trimmed)
	if err != nil {
		slog.Error("failed to create output file", "error", err, "path", trimmed)
		return NewStdoutWriter(format)
	}

	return &Writer{
		format: format,
		output: file,
	}
}

// NewStdoutWriter creates a new Writer that outputs to stdout in the specified format.
func NewStdoutWriter(format Format) *Writer {
	return &Writer{
		format: format,
		output: os.Stdout,
	}
}

// Serialize outputs the given configuration data in the configured format.
func (w *Writer) Serialize(config any) error {
	switch w.format {
	case FormatJSON:
		return w.serializeJSON(config)
	case FormatYAML:
		return w.serializeYAML(config)
	case FormatTable:
		return w.serializeTable(config)
	default:
		return fmt.Errorf("unsupported format: %s", w.format)
	}
}

func (w *Writer) serializeJSON(config any) error {
	encoder := json.NewEncoder(w.output)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(config); err != nil {
		return fmt.Errorf("failed to serialize to JSON: %w", err)
	}
	return nil
}

func (w *Writer) serializeYAML(config any) error {
	encoder := yaml.NewEncoder(w.output)
	encoder.SetIndent(2)
	if err := encoder.Encode(config); err != nil {
		return fmt.Errorf("failed to serialize to YAML: %w", err)
	}
	return nil
}

func (w *Writer) serializeTable(config any) error {
	flat := make(map[string]any)
	flattenValue(flat, reflect.ValueOf(config), "")
	if len(flat) == 0 {
		fmt.Fprintln(w.output, "<empty>")
		return nil
	}

	keys := make([]string, 0, len(flat))
	for k := range flat {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	tw := tabwriter.NewWriter(w.output, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "FIELD\tVALUE")
	fmt.Fprintln(tw, "-----\t-----")
	for _, key := range keys {
		fmt.Fprintf(tw, "%s\t%v\n", key, flat[key])
	}
	return tw.Flush()
}

func flattenValue(out map[string]any, val reflect.Value, prefix string) {
	if !val.IsValid() {
		return
	}

	for val.Kind() == reflect.Pointer || val.Kind() == reflect.Interface {
		if val.IsNil() {
			if prefix != "" {
				out[prefix] = nil
			}
			return
		}
		val = val.Elem()
	}

	switch val.Kind() {
	case reflect.Struct:
		typ := val.Type()
		for i := 0; i < val.NumField(); i++ {
			field := typ.Field(i)
			if !field.IsExported() {
				continue
			}
			key := joinKey(prefix, field.Name)
			flattenValue(out, val.Field(i), key)
		}
	case reflect.Map:
		for _, mapKey := range val.MapKeys() {
			key := joinKey(prefix, fmt.Sprintf("%v", mapKey.Interface()))
			flattenValue(out, val.MapIndex(mapKey), key)
		}
	case reflect.Slice, reflect.Array:
		for i := 0; i < val.Len(); i++ {
			key := joinKey(prefix, fmt.Sprintf("[%d]", i))
			flattenValue(out, val.Index(i), key)
		}
	default:
		if prefix == "" {
			prefix = "value"
		}
		out[prefix] = val.Interface()
	}
}

func joinKey(prefix, suffix string) string {
	if prefix == "" {
		return suffix
	}
	if suffix == "" {
		return prefix
	}
	return prefix + "." + suffix
}
