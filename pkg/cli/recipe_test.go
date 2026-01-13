/*
Copyright © 2025 NVIDIA Corporation
SPDX-License-Identifier: Apache-2.0
*/
package cli

import (
	"context"
	"strings"
	"testing"

	"github.com/urfave/cli/v3"

	"github.com/NVIDIA/cloud-native-stack/pkg/measurement"
	"github.com/NVIDIA/cloud-native-stack/pkg/recipe"
	"github.com/NVIDIA/cloud-native-stack/pkg/snapshotter"
)

func TestBuildCriteriaFromCmd(t *testing.T) {
	tests := []struct {
		name      string
		args      []string
		wantError bool
		errMsg    string
		validate  func(*testing.T, *recipe.Criteria)
	}{
		{
			name: "valid service",
			args: []string{"cmd", "--service", "eks"},
			validate: func(t *testing.T, c *recipe.Criteria) {
				if c.Service != recipe.CriteriaServiceEKS {
					t.Errorf("Service = %v, want %v", c.Service, recipe.CriteriaServiceEKS)
				}
			},
		},
		{
			name:      "invalid service",
			args:      []string{"cmd", "--service", "invalid-service"},
			wantError: true,
			errMsg:    "invalid service type",
		},
		{
			name: "valid accelerator",
			args: []string{"cmd", "--accelerator", "h100"},
			validate: func(t *testing.T, c *recipe.Criteria) {
				if c.Accelerator != recipe.CriteriaAcceleratorH100 {
					t.Errorf("Accelerator = %v, want %v", c.Accelerator, recipe.CriteriaAcceleratorH100)
				}
			},
		},
		{
			name: "valid accelerator with gpu alias",
			args: []string{"cmd", "--gpu", "gb200"},
			validate: func(t *testing.T, c *recipe.Criteria) {
				if c.Accelerator != recipe.CriteriaAcceleratorGB200 {
					t.Errorf("Accelerator = %v, want %v", c.Accelerator, recipe.CriteriaAcceleratorGB200)
				}
			},
		},
		{
			name:      "invalid accelerator",
			args:      []string{"cmd", "--accelerator", "invalid-gpu"},
			wantError: true,
			errMsg:    "invalid accelerator type",
		},
		{
			name: "valid intent",
			args: []string{"cmd", "--intent", "training"},
			validate: func(t *testing.T, c *recipe.Criteria) {
				if c.Intent != recipe.CriteriaIntentTraining {
					t.Errorf("Intent = %v, want %v", c.Intent, recipe.CriteriaIntentTraining)
				}
			},
		},
		{
			name:      "invalid intent",
			args:      []string{"cmd", "--intent", "invalid-intent"},
			wantError: true,
			errMsg:    "invalid intent type",
		},
		{
			name: "valid os",
			args: []string{"cmd", "--os", "ubuntu"},
			validate: func(t *testing.T, c *recipe.Criteria) {
				if c.OS != recipe.CriteriaOSUbuntu {
					t.Errorf("OS = %v, want %v", c.OS, recipe.CriteriaOSUbuntu)
				}
			},
		},
		{
			name:      "invalid os",
			args:      []string{"cmd", "--os", "invalid-os"},
			wantError: true,
			errMsg:    "invalid os type",
		},
		{
			name: "valid nodes",
			args: []string{"cmd", "--nodes", "8"},
			validate: func(t *testing.T, c *recipe.Criteria) {
				if c.Nodes != 8 {
					t.Errorf("Nodes = %v, want 8", c.Nodes)
				}
			},
		},
		{
			name: "complete criteria",
			args: []string{
				"cmd",
				"--service", "gke",
				"--accelerator", "a100",
				"--intent", "inference",
				"--os", "cos",
				"--nodes", "16",
			},
			validate: func(t *testing.T, c *recipe.Criteria) {
				if c.Service != recipe.CriteriaServiceGKE {
					t.Errorf("Service = %v, want %v", c.Service, recipe.CriteriaServiceGKE)
				}
				if c.Accelerator != recipe.CriteriaAcceleratorA100 {
					t.Errorf("Accelerator = %v, want %v", c.Accelerator, recipe.CriteriaAcceleratorA100)
				}
				if c.Intent != recipe.CriteriaIntentInference {
					t.Errorf("Intent = %v, want %v", c.Intent, recipe.CriteriaIntentInference)
				}
				if c.OS != recipe.CriteriaOSCOS {
					t.Errorf("OS = %v, want %v", c.OS, recipe.CriteriaOSCOS)
				}
				if c.Nodes != 16 {
					t.Errorf("Nodes = %v, want 16", c.Nodes)
				}
			},
		},
		{
			name: "empty criteria is valid",
			args: []string{"cmd"},
			validate: func(t *testing.T, c *recipe.Criteria) {
				if c == nil {
					t.Error("expected non-nil criteria")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var capturedCriteria *recipe.Criteria
			var capturedErr error

			testCmd := &cli.Command{
				Name: "test",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "service"},
					&cli.StringFlag{Name: "accelerator", Aliases: []string{"gpu"}},
					&cli.StringFlag{Name: "intent"},
					&cli.StringFlag{Name: "os"},
					&cli.IntFlag{Name: "nodes"},
				},
				Action: func(ctx context.Context, cmd *cli.Command) error {
					capturedCriteria, capturedErr = buildCriteriaFromCmd(cmd)
					return capturedErr
				},
			}

			err := testCmd.Run(context.Background(), tt.args)

			if tt.wantError {
				if err == nil && capturedErr == nil {
					t.Error("expected error but got nil")
					return
				}
				errToCheck := err
				if capturedErr != nil {
					errToCheck = capturedErr
				}
				if tt.errMsg != "" && !strings.Contains(errToCheck.Error(), tt.errMsg) {
					t.Errorf("error = %v, want error containing %v", errToCheck, tt.errMsg)
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if capturedErr != nil {
				t.Errorf("unexpected captured error: %v", capturedErr)
				return
			}

			if capturedCriteria == nil {
				t.Error("expected non-nil criteria")
				return
			}

			if tt.validate != nil {
				tt.validate(t, capturedCriteria)
			}
		})
	}
}

func TestExtractCriteriaFromSnapshot(t *testing.T) {
	tests := []struct {
		name           string
		snapshot       *snapshotter.Snapshot
		validate       func(*testing.T, *recipe.Criteria)
		validateDetect func(*testing.T, *CriteriaDetection)
	}{
		{
			name:     "nil snapshot",
			snapshot: nil,
			validate: func(t *testing.T, c *recipe.Criteria) {
				if c == nil {
					t.Error("expected non-nil criteria")
				}
			},
			validateDetect: func(t *testing.T, d *CriteriaDetection) {
				if d == nil {
					t.Error("expected non-nil detection")
				}
			},
		},
		{
			name: "empty snapshot",
			snapshot: &snapshotter.Snapshot{
				Measurements: nil,
			},
			validate: func(t *testing.T, c *recipe.Criteria) {
				if c == nil {
					t.Error("expected non-nil criteria")
				}
			},
		},
		{
			name: "snapshot with K8s service",
			snapshot: &snapshotter.Snapshot{
				Measurements: []*measurement.Measurement{
					{
						Type: "K8s",
						Subtypes: []measurement.Subtype{
							{
								Name: "server",
								Data: map[string]measurement.Reading{
									"service": measurement.Str("eks"),
								},
							},
						},
					},
				},
			},
			validate: func(t *testing.T, c *recipe.Criteria) {
				if c.Service != recipe.CriteriaServiceEKS {
					t.Errorf("Service = %v, want %v", c.Service, recipe.CriteriaServiceEKS)
				}
			},
			validateDetect: func(t *testing.T, d *CriteriaDetection) {
				if d.Service == nil {
					t.Error("expected service detection")
					return
				}
				if d.Service.Value != "eks" {
					t.Errorf("Service detection value = %v, want eks", d.Service.Value)
				}
				if d.Service.Source != "K8s server.service field" {
					t.Errorf("Service detection source = %v, want 'K8s server.service field'", d.Service.Source)
				}
			},
		},
		{
			name: "snapshot with GPU H100",
			snapshot: &snapshotter.Snapshot{
				Measurements: []*measurement.Measurement{
					{
						Type: "GPU",
						Subtypes: []measurement.Subtype{
							{
								Name: "device",
								Data: map[string]measurement.Reading{
									"model": measurement.Str("NVIDIA H100 80GB HBM3"),
								},
							},
						},
					},
				},
			},
			validate: func(t *testing.T, c *recipe.Criteria) {
				if c.Accelerator != recipe.CriteriaAcceleratorH100 {
					t.Errorf("Accelerator = %v, want %v", c.Accelerator, recipe.CriteriaAcceleratorH100)
				}
			},
			validateDetect: func(t *testing.T, d *CriteriaDetection) {
				if d.Accelerator == nil {
					t.Error("expected accelerator detection")
					return
				}
				if d.Accelerator.Value != "h100" {
					t.Errorf("Accelerator detection value = %v, want h100", d.Accelerator.Value)
				}
				if d.Accelerator.RawValue != "NVIDIA H100 80GB HBM3" {
					t.Errorf("Accelerator raw value = %v, want 'NVIDIA H100 80GB HBM3'", d.Accelerator.RawValue)
				}
			},
		},
		{
			name: "snapshot with GB200",
			snapshot: &snapshotter.Snapshot{
				Measurements: []*measurement.Measurement{
					{
						Type: "GPU",
						Subtypes: []measurement.Subtype{
							{
								Name: "device",
								Data: map[string]measurement.Reading{
									"model": measurement.Str("NVIDIA GB200"),
								},
							},
						},
					},
				},
			},
			validate: func(t *testing.T, c *recipe.Criteria) {
				if c.Accelerator != recipe.CriteriaAcceleratorGB200 {
					t.Errorf("Accelerator = %v, want %v", c.Accelerator, recipe.CriteriaAcceleratorGB200)
				}
			},
		},
		{
			name: "snapshot with OS ubuntu",
			snapshot: &snapshotter.Snapshot{
				Measurements: []*measurement.Measurement{
					{
						Type: "OS",
						Subtypes: []measurement.Subtype{
							{
								Name: "release",
								Data: map[string]measurement.Reading{
									"ID": measurement.Str("ubuntu"),
								},
							},
						},
					},
				},
			},
			validate: func(t *testing.T, c *recipe.Criteria) {
				if c.OS != recipe.CriteriaOSUbuntu {
					t.Errorf("OS = %v, want %v", c.OS, recipe.CriteriaOSUbuntu)
				}
			},
			validateDetect: func(t *testing.T, d *CriteriaDetection) {
				if d.OS == nil {
					t.Error("expected OS detection")
					return
				}
				if d.OS.Source != "/etc/os-release ID" {
					t.Errorf("OS detection source = %v, want '/etc/os-release ID'", d.OS.Source)
				}
			},
		},
		{
			name: "complete snapshot",
			snapshot: &snapshotter.Snapshot{
				Measurements: []*measurement.Measurement{
					{
						Type: "K8s",
						Subtypes: []measurement.Subtype{
							{
								Name: "server",
								Data: map[string]measurement.Reading{
									"service": measurement.Str("gke"),
								},
							},
						},
					},
					{
						Type: "GPU",
						Subtypes: []measurement.Subtype{
							{
								Name: "device",
								Data: map[string]measurement.Reading{
									"model": measurement.Str("A100-SXM4-80GB"),
								},
							},
						},
					},
					{
						Type: "OS",
						Subtypes: []measurement.Subtype{
							{
								Name: "release",
								Data: map[string]measurement.Reading{
									"ID": measurement.Str("rhel"),
								},
							},
						},
					},
				},
			},
			validate: func(t *testing.T, c *recipe.Criteria) {
				if c.Service != recipe.CriteriaServiceGKE {
					t.Errorf("Service = %v, want %v", c.Service, recipe.CriteriaServiceGKE)
				}
				if c.Accelerator != recipe.CriteriaAcceleratorA100 {
					t.Errorf("Accelerator = %v, want %v", c.Accelerator, recipe.CriteriaAcceleratorA100)
				}
				if c.OS != recipe.CriteriaOSRHEL {
					t.Errorf("OS = %v, want %v", c.OS, recipe.CriteriaOSRHEL)
				}
			},
		},
		{
			name: "snapshot with K8s version string EKS",
			snapshot: &snapshotter.Snapshot{
				Measurements: []*measurement.Measurement{
					{
						Type: "K8s",
						Subtypes: []measurement.Subtype{
							{
								Name: "server",
								Data: map[string]measurement.Reading{
									"version": measurement.Str("v1.33.5-eks-3025e55"),
								},
							},
						},
					},
				},
			},
			validate: func(t *testing.T, c *recipe.Criteria) {
				if c.Service != recipe.CriteriaServiceEKS {
					t.Errorf("Service = %v, want %v", c.Service, recipe.CriteriaServiceEKS)
				}
			},
			validateDetect: func(t *testing.T, d *CriteriaDetection) {
				if d.Service == nil {
					t.Error("expected service detection")
					return
				}
				if d.Service.Source != "K8s version string" {
					t.Errorf("Service detection source = %v, want 'K8s version string'", d.Service.Source)
				}
				if d.Service.RawValue != "v1.33.5-eks-3025e55" {
					t.Errorf("Service raw value = %v, want 'v1.33.5-eks-3025e55'", d.Service.RawValue)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			criteria, detection := extractCriteriaFromSnapshot(tt.snapshot)

			if tt.validate != nil {
				tt.validate(t, criteria)
			}
			if tt.validateDetect != nil {
				tt.validateDetect(t, detection)
			}
		})
	}
}

func TestApplyCriteriaOverrides(t *testing.T) {
	tests := []struct {
		name           string
		args           []string
		initial        *recipe.Criteria
		initialDetect  *CriteriaDetection
		validate       func(*testing.T, *recipe.Criteria)
		validateDetect func(*testing.T, *CriteriaDetection)
		wantErr        bool
	}{
		{
			name:          "override service",
			args:          []string{"cmd", "--service", "aks"},
			initial:       &recipe.Criteria{Service: recipe.CriteriaServiceEKS},
			initialDetect: &CriteriaDetection{Service: &DetectionSource{Value: "eks", Source: "test"}},
			validate: func(t *testing.T, c *recipe.Criteria) {
				if c.Service != recipe.CriteriaServiceAKS {
					t.Errorf("Service = %v, want %v", c.Service, recipe.CriteriaServiceAKS)
				}
			},
			validateDetect: func(t *testing.T, d *CriteriaDetection) {
				if !d.Service.Overridden {
					t.Error("Service should be marked as overridden")
				}
			},
		},
		{
			name:          "override accelerator",
			args:          []string{"cmd", "--accelerator", "l40"},
			initial:       &recipe.Criteria{Accelerator: recipe.CriteriaAcceleratorH100},
			initialDetect: &CriteriaDetection{Accelerator: &DetectionSource{Value: "h100", Source: "test"}},
			validate: func(t *testing.T, c *recipe.Criteria) {
				if c.Accelerator != recipe.CriteriaAcceleratorL40 {
					t.Errorf("Accelerator = %v, want %v", c.Accelerator, recipe.CriteriaAcceleratorL40)
				}
			},
			validateDetect: func(t *testing.T, d *CriteriaDetection) {
				if !d.Accelerator.Overridden {
					t.Error("Accelerator should be marked as overridden")
				}
			},
		},
		{
			name:          "no overrides preserves existing",
			args:          []string{"cmd"},
			initial:       &recipe.Criteria{Service: recipe.CriteriaServiceGKE, Accelerator: recipe.CriteriaAcceleratorGB200},
			initialDetect: &CriteriaDetection{},
			validate: func(t *testing.T, c *recipe.Criteria) {
				if c.Service != recipe.CriteriaServiceGKE {
					t.Errorf("Service = %v, want %v", c.Service, recipe.CriteriaServiceGKE)
				}
				if c.Accelerator != recipe.CriteriaAcceleratorGB200 {
					t.Errorf("Accelerator = %v, want %v", c.Accelerator, recipe.CriteriaAcceleratorGB200)
				}
			},
		},
		{
			name:          "invalid override returns error",
			args:          []string{"cmd", "--service", "invalid"},
			initial:       &recipe.Criteria{},
			initialDetect: &CriteriaDetection{},
			wantErr:       true,
		},
		{
			name:          "set intent from flag when not auto-detected",
			args:          []string{"cmd", "--intent", "training"},
			initial:       &recipe.Criteria{},
			initialDetect: &CriteriaDetection{},
			validate: func(t *testing.T, c *recipe.Criteria) {
				if c.Intent != recipe.CriteriaIntentTraining {
					t.Errorf("Intent = %v, want %v", c.Intent, recipe.CriteriaIntentTraining)
				}
			},
			validateDetect: func(t *testing.T, d *CriteriaDetection) {
				if d.Intent == nil {
					t.Error("expected intent detection")
					return
				}
				if d.Intent.Source != "--intent flag" {
					t.Errorf("Intent source = %v, want '--intent flag'", d.Intent.Source)
				}
				if d.Intent.Overridden {
					t.Error("Intent should NOT be marked as overridden (was not previously detected)")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testCmd := &cli.Command{
				Name: "test",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "service"},
					&cli.StringFlag{Name: "accelerator", Aliases: []string{"gpu"}},
					&cli.StringFlag{Name: "intent"},
					&cli.StringFlag{Name: "os"},
					&cli.IntFlag{Name: "nodes"},
				},
				Action: func(ctx context.Context, cmd *cli.Command) error {
					return applyCriteriaOverrides(cmd, tt.initial, tt.initialDetect)
				},
			}

			err := testCmd.Run(context.Background(), tt.args)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error but got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if tt.validate != nil {
				tt.validate(t, tt.initial)
			}
			if tt.validateDetect != nil {
				tt.validateDetect(t, tt.initialDetect)
			}
		})
	}
}

func TestRecipeCmd_CommandStructure(t *testing.T) {
	cmd := recipeCmd()

	if cmd.Name != "recipe" {
		t.Errorf("Name = %v, want recipe", cmd.Name)
	}

	if cmd.Usage == "" {
		t.Error("Usage should not be empty")
	}

	if cmd.Description == "" {
		t.Error("Description should not be empty")
	}

	requiredFlags := []string{"service", "accelerator", "intent", "os", "nodes", "snapshot", "output", "format"}
	for _, flagName := range requiredFlags {
		found := false
		for _, flag := range cmd.Flags {
			if hasName(flag, flagName) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("required flag %q not found", flagName)
		}
	}

	if cmd.Action == nil {
		t.Error("Action should not be nil")
	}
}

func TestSnapshotCmd_CommandStructure(t *testing.T) {
	cmd := snapshotCmd()

	if cmd.Name != "snapshot" {
		t.Errorf("Name = %v, want snapshot", cmd.Name)
	}

	if cmd.Usage == "" {
		t.Error("Usage should not be empty")
	}

	if cmd.Description == "" {
		t.Error("Description should not be empty")
	}

	requiredFlags := []string{"output", "format"}
	for _, flagName := range requiredFlags {
		found := false
		for _, flag := range cmd.Flags {
			if hasName(flag, flagName) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("required flag %q not found", flagName)
		}
	}

	if cmd.Action == nil {
		t.Error("Action should not be nil")
	}
}

func TestCommandLister(_ *testing.T) {
	commandLister(context.Background(), nil)

	cmd := &cli.Command{Name: "test"}
	commandLister(context.Background(), cmd)

	rootCmd := &cli.Command{
		Name: "root",
		Commands: []*cli.Command{
			{Name: "visible1", Hidden: false},
			{Name: "hidden", Hidden: true},
			{Name: "visible2", Hidden: false},
		},
	}
	commandLister(context.Background(), rootCmd)
}

func TestContainsIgnoreCase(t *testing.T) {
	tests := []struct {
		s      string
		substr string
		want   bool
	}{
		{"NVIDIA H100", "h100", true},
		{"h100", "H100", true},
		{"GB200", "gb200", true},
		{"NVIDIA A100-SXM4-80GB", "a100", true},
		{"L40S", "l40", true},
		{"H100", "gb200", false},
		{"", "h100", false},
		{"h100", "", true}, // empty substr matches anything
		{"", "", true},     // empty matches empty
	}

	for _, tt := range tests {
		t.Run(tt.s+"_"+tt.substr, func(t *testing.T) {
			got := containsIgnoreCase(tt.s, tt.substr)
			if got != tt.want {
				t.Errorf("containsIgnoreCase(%q, %q) = %v, want %v", tt.s, tt.substr, got, tt.want)
			}
		})
	}
}

func TestPrintDetection(t *testing.T) {
	tests := []struct {
		name      string
		detection *CriteriaDetection
		wantLines []string
	}{
		{
			name:      "empty detection",
			detection: &CriteriaDetection{},
			wantLines: []string{
				"Detected criteria from snapshot:",
				"service:     (not detected)",
				"accelerator: (not detected)",
				"os:          (not detected)",
				"intent:      (not detected)",
				"nodes:       (not detected)",
			},
		},
		{
			name: "detected service from version string",
			detection: &CriteriaDetection{
				Service: &DetectionSource{
					Value:    "eks",
					Source:   "K8s version string",
					RawValue: "v1.33.5-eks-3025e55",
				},
			},
			wantLines: []string{
				"service:     eks          (from K8s version string: v1.33.5-eks-3025e55)",
			},
		},
		{
			name: "detected accelerator",
			detection: &CriteriaDetection{
				Accelerator: &DetectionSource{
					Value:    "h100",
					Source:   "nvidia-smi gpu.model",
					RawValue: "NVIDIA H100 80GB HBM3",
				},
			},
			wantLines: []string{
				"accelerator: h100         (from nvidia-smi gpu.model: NVIDIA H100 80GB HBM3)",
			},
		},
		{
			name: "overridden value",
			detection: &CriteriaDetection{
				Service: &DetectionSource{
					Value:      "gke",
					Source:     "--service flag",
					Overridden: true,
				},
			},
			wantLines: []string{
				"service:     gke          (overridden by --service flag)",
			},
		},
		{
			name: "value equals raw value",
			detection: &CriteriaDetection{
				OS: &DetectionSource{
					Value:    "ubuntu",
					Source:   "/etc/os-release ID",
					RawValue: "ubuntu",
				},
			},
			wantLines: []string{
				"os:          ubuntu       (from /etc/os-release ID)",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf strings.Builder
			tt.detection.PrintDetection(&buf)
			output := buf.String()

			for _, line := range tt.wantLines {
				if !strings.Contains(output, line) {
					t.Errorf("output missing expected line %q\nGot:\n%s", line, output)
				}
			}
		})
	}
}

func TestPrintDetectionField(t *testing.T) {
	tests := []struct {
		name   string
		field  string
		source *DetectionSource
		want   string
	}{
		{
			name:   "nil source",
			field:  "service",
			source: nil,
			want:   "service:     (not detected)",
		},
		{
			name:  "simple detected value",
			field: "os",
			source: &DetectionSource{
				Value:  "ubuntu",
				Source: "/etc/os-release ID",
			},
			want: "os:          ubuntu       (from /etc/os-release ID)",
		},
		{
			name:  "detected with different raw value",
			field: "accelerator",
			source: &DetectionSource{
				Value:    "h100",
				Source:   "nvidia-smi",
				RawValue: "NVIDIA H100 80GB",
			},
			want: "accelerator: h100         (from nvidia-smi: NVIDIA H100 80GB)",
		},
		{
			name:  "overridden value",
			field: "intent",
			source: &DetectionSource{
				Value:      "training",
				Source:     "--intent flag",
				Overridden: true,
			},
			want: "intent:      training     (overridden by --intent flag)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf strings.Builder
			printDetectionField(&buf, tt.field, tt.source)
			got := strings.TrimSpace(buf.String())
			want := strings.TrimSpace(tt.want)

			if got != want {
				t.Errorf("printDetectionField() =\n%q\nwant:\n%q", got, want)
			}
		})
	}
}

func hasName(flag cli.Flag, name string) bool {
	if flag == nil {
		return false
	}
	names := flag.Names()
	for _, n := range names {
		if n == name {
			return true
		}
	}
	return false
}
