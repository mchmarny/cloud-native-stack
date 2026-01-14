package systemd

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/NVIDIA/cloud-native-stack/pkg/measurement"
	"github.com/coreos/go-systemd/v22/dbus"
)

var (
	// Keys to filter out from systemd properties for privacy/security or noise reduction
	filterOutSystemDKeys = []string{
		"AllowedCPUs",
		"AllowedMemoryNodes",
		"Asserts",
		"BPFProgram",
		"BusName",
		"Id",
		"*Credential*",
	}
)

// Collector is a collector that gathers configuration data from systemd services.
type Collector struct {
	Services []string
}

// Collect gathers configuration data from specified systemd services.
// It implements the Collector interface.
func (s *Collector) Collect(ctx context.Context) (*measurement.Measurement, error) {
	slog.Info("collecting SystemD service configurations")

	services := s.Services
	if len(services) == 0 {
		services = []string{"containerd.service"}
	}
	subs := make([]measurement.Subtype, 0)

	conn, err := dbus.NewSystemdConnectionContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to systemd: %w", err)
	}
	defer conn.Close()

	for _, service := range services {
		data, err := conn.GetAllPropertiesContext(ctx, service)
		if err != nil {
			return nil, fmt.Errorf("failed to get unit properties: %w", err)
		}

		readings := make(map[string]measurement.Reading)
		for k, v := range data {
			readings[k] = measurement.ToReading(v)
		}

		subs = append(subs, measurement.Subtype{
			Name: service,
			Data: measurement.FilterOut(readings, filterOutSystemDKeys),
		})
	}

	res := &measurement.Measurement{
		Type:     measurement.TypeSystemD,
		Subtypes: subs,
	}

	return res, nil
}
