package collectors

import (
	"context"
	"fmt"

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
func (s *SystemDCollector) Collect(ctx context.Context) ([]Configuration, error) {
	services := s.Services
	if len(services) == 0 {
		services = []string{"containerd.service"}
	}
	res := make([]Configuration, 0, len(services)*10)

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

		res = append(res, Configuration{
			Type: SystemDType,
			Data: SystemDConfig{
				Unit:       service,
				Properties: data,
			},
		})
	}

	return res, nil
}
