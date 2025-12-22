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

// SystemDType is the type identifier for systemd configurations.
const SystemDType string = "SystemD"

// SystemDConfig represents the configuration data collected from a systemd service.
type SystemDConfig struct {
	Unit       string         `json:"unit" yaml:"unit"`
	Properties map[string]any `json:"properties" yaml:"properties"`
}

// Collect gathers configuration data from specified systemd services.
// It implements the Collector interface.
func (s *SystemDCollector) Collect(ctx context.Context) ([]measurement.Measurement, error) {
	services := s.Services
	if len(services) == 0 {
		services = []string{"containerd.service"}
	}
	res := make([]measurement.Measurement, 0, len(services)*10)

	conn, err := dbus.NewSystemdConnectionContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to systemd: %w", err)
	}
	defer conn.Close()

	list := make([]SystemDConfig, 0)

	for _, service := range services {
		data, err := conn.GetAllPropertiesContext(ctx, service)
		if err != nil {
			return nil, fmt.Errorf("failed to get unit properties: %w", err)
		}

		list = append(list, SystemDConfig{
			Unit:       service,
			Properties: data,
		})
	}

	res = append(res, measurement.Measurement{
		Type: SystemDType,
		Data: list,
	})

	return res, nil
}
