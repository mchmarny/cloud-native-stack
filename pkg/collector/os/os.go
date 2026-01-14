package os

import (
	"context"
	"log/slog"

	"github.com/NVIDIA/cloud-native-stack/pkg/measurement"
)

// Collector collects operating system configuration including:
// - GRUB bootloader parameters from /proc/cmdline
// - Loaded kernel modules from /proc/modules
// - Sysctl parameters from /proc/sys
type Collector struct {
}

// Collect gathers all OS-level configurations and returns them as a single measurement
// with three subtypes: grub, kmod, and sysctl.
func (c *Collector) Collect(ctx context.Context) (*measurement.Measurement, error) {
	slog.Info("collecting OS configuration")

	// Check if context is canceled
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	grub, err := c.collectGRUB(ctx)
	if err != nil {
		return nil, err
	}

	sysctl, err := c.collectSysctl(ctx)
	if err != nil {
		return nil, err
	}

	kmod, err := c.collectKMod(ctx)
	if err != nil {
		return nil, err
	}

	release, err := c.collectRelease(ctx)
	if err != nil {
		return nil, err
	}

	res := &measurement.Measurement{
		Type: measurement.TypeOS,
		Subtypes: []measurement.Subtype{
			*grub,
			*sysctl,
			*kmod,
			*release,
		},
	}

	return res, nil
}
