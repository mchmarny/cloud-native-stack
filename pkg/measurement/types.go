package measurement

// Measurement represents a single collector configuration measurement.
type Measurement struct {
	Type string `json:"type" yaml:"type"`
	Data any    `json:"data" yaml:"data"`
}
