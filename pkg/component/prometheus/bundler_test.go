package prometheus

import (
	"testing"

	"github.com/NVIDIA/cloud-native-stack/pkg/bundler/config"
	"github.com/stretchr/testify/require"
)

func TestNewBundler(t *testing.T) {
	tests := []struct {
		name   string
		config *config.Config
	}{
		{
			name:   "nil config",
			config: nil,
		},
		{
			name:   "valid config",
			config: &config.Config{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bundler := NewBundler(tt.config)
			require.NotNil(t, bundler)
			require.NotNil(t, bundler.BaseBundler)
		})
	}
}