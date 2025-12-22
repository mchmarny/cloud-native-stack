package collectors

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/NVIDIA/cloud-native-stack/pkg/measurement"
)

// SysctlCollector collects sysctl configurations from /proc/sys
// excluding /proc/sys/net
type SysctlCollector struct {
}

// SysctlType is the type identifier for sysctl configurations
const SysctlType string = "Sysctl"

// Collect gathers sysctl configurations from /proc/sys, excluding /proc/sys/net
// and returns them as a single Configuration with a map of all parameters.
func (s *SysctlCollector) Collect(ctx context.Context) ([]measurement.Measurement, error) {
	root := "/proc/sys"
	params := make(map[string]any)

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return fmt.Errorf("failed to walk dir: %w", err)
		}

		// Check if context is canceled
		if ctx.Err() != nil {
			return ctx.Err()
		}

		// Skip symlinks to prevent directory traversal attacks
		if d.Type()&fs.ModeSymlink != 0 {
			return nil
		}

		if d.IsDir() {
			return nil
		}

		// Ensure path is under root (defense in depth)
		if !strings.HasPrefix(path, root) {
			return fmt.Errorf("path traversal detected: %s", path)
		}

		if strings.HasPrefix(path, "/proc/sys/net") {
			return nil
		}

		c, err := os.ReadFile(path)
		if err != nil {
			// Skip files we can't read (some proc files are write-only or restricted)
			return nil
		}

		params[path] = strings.TrimSpace(string(c))

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to capture sysctl config: %w", err)
	}

	res := []measurement.Measurement{
		{
			Type: SysctlType,
			Data: params,
		},
	}

	return res, nil
}
