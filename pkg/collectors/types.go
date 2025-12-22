package collectors

import (
	"context"

	"github.com/NVIDIA/cloud-native-stack/pkg/measurement"
)

// Collector is an interface for collecting measurement data.
// Implementations of this interface can collect data from various sources
// such as system modules, services, or settings.
type Collector interface {
	Collect(ctx context.Context) (*measurement.Measurement, error)
}
