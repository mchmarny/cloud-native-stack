package collectors

import "context"

// Collector is an interface for collecting measurement data.
// Implementations of this interface can collect data from various sources
// such as system modules, services, or settings.
type Collector interface {
	Collect(ctx context.Context) ([]Measurement, error)
}

// Measurement represents a single collector configuration measurement.
type Measurement struct {
	Type string `json:"type" yaml:"type"`
	Data any    `json:"data" yaml:"data"`
}
