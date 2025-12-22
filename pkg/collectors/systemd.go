package collectors

import (
	"context"
	"fmt"

	"github.com/NVIDIA/cloud-native-stack/pkg/measurement"
	"github.com/coreos/go-systemd/v22/dbus"
)

// SystemDCollector is a collector that gathers configuration data from systemd services.
type SystemDCollector struct {
	Services []string
}

// Collect gathers configuration data from specified systemd services.
// It implements the Collector interface.
func (s *SystemDCollector) Collect(ctx context.Context) (*measurement.Measurement, error) {
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
			Data: readings,
		})
	}

	res := &measurement.Measurement{
		Type:     measurement.TypeSystemD,
		Subtypes: subs,
	}

	return res, nil
}
