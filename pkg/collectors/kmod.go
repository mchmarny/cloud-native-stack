package collectors

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/NVIDIA/cloud-native-stack/pkg/measurement"
)

// KModCollector collects information about loaded kernel modules from /proc/modules
// and parses them into KModConfig structures
type KModCollector struct {
}

// KModType is the type identifier for kernel module configurations
const KModType string = "KMod"

// Collect retrieves the list of loaded kernel modules from /proc/modules
// and parses them into KModConfig structures
func (s *KModCollector) Collect(ctx context.Context) ([]measurement.Measurement, error) {
	// Check if context is canceled
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	root := "/proc/modules"
	modules := make([]string, 0, 100)

	cmdline, err := os.ReadFile(root)
	if err != nil {
		return nil, fmt.Errorf("failed to read KMod config: %w", err)
	}

	params := strings.Split(string(cmdline), "\n")

	for _, param := range params {
		p := strings.TrimSpace(param)
		if p == "" {
			continue
		}

		mod := strings.Split(p, " ")
		modules = append(modules, mod[0])
	}

	res := []measurement.Measurement{
		{
			Type: KModType,
			Data: modules,
		},
	}

	return res, nil
}
