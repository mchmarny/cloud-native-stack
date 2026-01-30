// Copyright (c) 2025, NVIDIA CORPORATION.  All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package snapshotter

import (
	"os"
	"testing"

	corev1 "k8s.io/api/core/v1"
)

func TestLogWriter(t *testing.T) {
	writer := logWriter()
	if writer == nil {
		t.Fatal("logWriter() returned nil")
	}
	if writer != os.Stderr {
		t.Errorf("logWriter() = %v, want os.Stderr", writer)
	}
}

func TestDefaultTolerations(t *testing.T) {
	tolerations := DefaultTolerations()

	if len(tolerations) != 1 {
		t.Fatalf("DefaultTolerations() returned %d tolerations, want 1", len(tolerations))
	}

	tol := tolerations[0]
	if tol.Operator != corev1.TolerationOpExists {
		t.Errorf("DefaultTolerations()[0].Operator = %v, want %v", tol.Operator, corev1.TolerationOpExists)
	}
	if tol.Key != "" {
		t.Errorf("DefaultTolerations()[0].Key = %q, want empty string", tol.Key)
	}
}

func TestAgentConfig_Defaults(t *testing.T) {
	// Test that AgentConfig can be instantiated with zero values
	cfg := AgentConfig{}

	if cfg.Enabled {
		t.Error("AgentConfig.Enabled should default to false")
	}
	if cfg.Cleanup {
		t.Error("AgentConfig.Cleanup should default to false")
	}
	if cfg.Debug {
		t.Error("AgentConfig.Debug should default to false")
	}
	if cfg.Privileged {
		t.Error("AgentConfig.Privileged should default to false")
	}
	if cfg.Timeout != 0 {
		t.Errorf("AgentConfig.Timeout should default to 0, got %v", cfg.Timeout)
	}
}
