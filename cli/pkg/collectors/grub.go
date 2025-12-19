package collectors

import (
	"context"
	"fmt"
	"os"
	"strings"
	"unicode/utf8"
)

// GrubCollector collects information about GRUB bootloader configurations from /proc/cmdline
// and parses them into GrubConfig structures
type GrubCollector struct {
}

// GrubType is the type identifier for GRUB configurations
const GrubType string = "Grub"

// GrubConfig represents a single GRUB bootloader configuration parameter
// with its key and value
type GrubConfig struct {
	Key   string
	Value string
}

// Collect retrieves the GRUB bootloader parameters from /proc/cmdline
// and parses them into GrubConfig structures
func (s *GrubCollector) Collect(ctx context.Context) ([]Configuration, error) {
	// Check if context is canceled
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	root := "/proc/cmdline"
	res := make([]Configuration, 0, 20)

	cmdline, err := os.ReadFile(root)
	if err != nil {
		return nil, fmt.Errorf("failed to read grub config: %w", err)
	}

	// Validate UTF-8
	if !utf8.Valid(cmdline) {
		return nil, fmt.Errorf("grub config contains invalid UTF-8")
	}

	// Limit size (1MB max)
	const maxSize = 1 << 20
	if len(cmdline) > maxSize {
		return nil, fmt.Errorf("grub config exceeds maximum size of %d bytes", maxSize)
	}

	params := strings.Split(string(cmdline), " ")

	for _, param := range params {
		p := strings.TrimSpace(param)
		if p == "" {
			continue
		}

		key, val := "", ""
		// Split on first '=' only to handle values like "root=PARTUUID=xyz"
		s := strings.SplitN(p, "=", 2)
		if len(s) == 1 {
			key = s[0]
		} else {
			key = s[0]
			val = s[1]
		}

		res = append(res, Configuration{
			Type: GrubType,
			Data: GrubConfig{
				Key:   key,
				Value: val,
			},
		})
	}

	return res, nil
}
