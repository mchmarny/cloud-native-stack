package serializer

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// FormatFromPath determines the format based on file extension.
// Returns FormatJSON as default if the extension is not recognized.
func FormatFromPath(filePath string) Format {
	lowerPath := strings.ToLower(filePath)
	switch {
	case strings.HasSuffix(lowerPath, ".json"):
		return FormatJSON
	case strings.HasSuffix(lowerPath, ".yaml"), strings.HasSuffix(lowerPath, ".yml"):
		return FormatYAML
	case strings.HasSuffix(lowerPath, ".table"), strings.HasSuffix(lowerPath, ".txt"):
		return FormatTable
	default:
		slog.Warn("unknown file extension, defaulting to JSON", "filePath", filePath)
		return FormatJSON
	}
}

// Reader handles deserialization of configuration data from various formats.
// Close must be called to release resources when using NewFileReader.
type Reader struct {
	format Format
	input  io.Reader
	closer io.Closer
}

// NewReader creates a new Reader with the specified format and input source.
// If format is unknown, returns an error.
// The caller is responsible for closing the input if it implements io.Closer.
func NewReader(format Format, input io.Reader) (*Reader, error) {
	if format.IsUnknown() {
		return nil, fmt.Errorf("unknown format: %s", format)
	}

	if format == FormatTable {
		return nil, fmt.Errorf("table format does not support deserialization")
	}

	r := &Reader{
		format: format,
		input:  input,
	}

	// Store closer if input implements it
	if closer, ok := input.(io.Closer); ok {
		r.closer = closer
	}

	return r, nil
}

// NewFileReader creates a new Reader that reads from the specified file path.
// The format is determined by the provided format parameter.
// Close must be called to release the file handle.
func NewFileReader(format Format, filePath string) (*Reader, error) {
	if format.IsUnknown() {
		return nil, fmt.Errorf("unknown format: %s", format)
	}

	if format == FormatTable {
		return nil, fmt.Errorf("table format does not support deserialization")
	}

	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}

	return &Reader{
		format: format,
		input:  file,
		closer: file,
	}, nil
}

// NewFileReaderAuto creates a new Reader that reads from the specified file path.
// The format is automatically determined from the file extension.
// Close must be called to release the file handle.
func NewFileReaderAuto(filePath string) (*Reader, error) {
	format := FormatFromPath(filePath)
	return NewFileReader(format, filePath)
}

// Deserialize reads data from the input source and unmarshals it into v.
// The type of v must be a pointer to the target structure.
func (r *Reader) Deserialize(v any) error {
	if r == nil {
		return fmt.Errorf("reader is nil")
	}

	if r.input == nil {
		return fmt.Errorf("input source is nil")
	}

	switch r.format {
	case FormatJSON:
		decoder := json.NewDecoder(r.input)
		if err := decoder.Decode(v); err != nil {
			return fmt.Errorf("failed to decode JSON: %w", err)
		}
		return nil

	case FormatYAML:
		decoder := yaml.NewDecoder(r.input)
		if err := decoder.Decode(v); err != nil {
			return fmt.Errorf("failed to decode YAML: %w", err)
		}
		return nil

	case FormatTable:
		return fmt.Errorf("table format is not supported for deserialization")

	default:
		return fmt.Errorf("unsupported format for deserialization: %s", r.format)
	}
}

// Close releases any resources held by the Reader.
// It is safe to call Close multiple times.
func (r *Reader) Close() error {
	if r == nil {
		return nil
	}

	if r.closer != nil {
		err := r.closer.Close()
		r.closer = nil // Prevent double-close
		return err
	}
	return nil
}
