package collectors

import "context"

// Collector is an interface for collecting configuration data.
// Implementations of this interface can collect data from various sources
// such as system modules, services, or settings.
type Collector interface {
	Collect(ctx context.Context) ([]Configuration, error)
}

// Configuration represents a single collector configuration.
type Configuration struct {
	Type string `json:"type" yaml:"type"`
	Data any    `json:"data" yaml:"data"`
}
