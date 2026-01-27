package result

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/NVIDIA/cloud-native-stack/pkg/bundler/types"
)

// TestOutput_New tests Output initialization
func TestOutput_New(t *testing.T) {
	outputDir := "/output/bundles"
	output := &Output{
		OutputDir: outputDir,
		Results:   []*Result{},
		Errors:    []BundleError{},
	}

	if output.OutputDir != outputDir {
		t.Errorf("OutputDir = %s, want %s", output.OutputDir, outputDir)
	}

	if output.Results == nil {
		t.Error("Results slice should not be nil")
	}

	if output.Errors == nil {
		t.Error("Errors slice should not be nil")
	}

	if output.TotalSize != 0 {
		t.Errorf("TotalSize = %d, want 0", output.TotalSize)
	}

	if output.TotalFiles != 0 {
		t.Errorf("TotalFiles = %d, want 0", output.TotalFiles)
	}

	if output.TotalDuration != 0 {
		t.Errorf("TotalDuration = %v, want 0", output.TotalDuration)
	}
}

// TestOutput_HasErrors tests error detection
func TestOutput_HasErrors(t *testing.T) {
	tests := []struct {
		name   string
		errors []BundleError
		want   bool
	}{
		{
			name:   "no errors",
			errors: []BundleError{},
			want:   false,
		},
		{
			name: "single error",
			errors: []BundleError{
				{BundlerType: types.BundleType("gpu-operator"), Error: "failed"},
			},
			want: true,
		},
		{
			name: "multiple errors",
			errors: []BundleError{
				{BundlerType: types.BundleType("gpu-operator"), Error: "error 1"},
				{BundlerType: types.BundleType("network-operator"), Error: "error 2"},
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := &Output{
				Errors: tt.errors,
			}

			got := output.HasErrors()
			if got != tt.want {
				t.Errorf("HasErrors() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestOutput_SuccessCount tests counting successful results
func TestOutput_SuccessCount(t *testing.T) {
	tests := []struct {
		name    string
		results []*Result
		want    int
	}{
		{
			name:    "no results",
			results: []*Result{},
			want:    0,
		},
		{
			name: "all successful",
			results: []*Result{
				{Type: types.BundleType("gpu-operator"), Success: true},
				{Type: types.BundleType("network-operator"), Success: true},
			},
			want: 2,
		},
		{
			name: "all failed",
			results: []*Result{
				{Type: types.BundleType("gpu-operator"), Success: false},
				{Type: types.BundleType("network-operator"), Success: false},
			},
			want: 0,
		},
		{
			name: "mixed success and failure",
			results: []*Result{
				{Type: types.BundleType("gpu-operator"), Success: true},
				{Type: types.BundleType("network-operator"), Success: false},
				{Type: types.BundleType("custom"), Success: true},
			},
			want: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := &Output{
				Results: tt.results,
			}

			got := output.SuccessCount()
			if got != tt.want {
				t.Errorf("SuccessCount() = %d, want %d", got, tt.want)
			}
		})
	}
}

// TestOutput_FailureCount tests counting failed results
func TestOutput_FailureCount(t *testing.T) {
	tests := []struct {
		name    string
		results []*Result
		want    int
	}{
		{
			name:    "no results",
			results: []*Result{},
			want:    0,
		},
		{
			name: "all successful",
			results: []*Result{
				{Type: types.BundleType("gpu-operator"), Success: true},
				{Type: types.BundleType("network-operator"), Success: true},
			},
			want: 0,
		},
		{
			name: "all failed",
			results: []*Result{
				{Type: types.BundleType("gpu-operator"), Success: false},
				{Type: types.BundleType("network-operator"), Success: false},
			},
			want: 2,
		},
		{
			name: "mixed success and failure",
			results: []*Result{
				{Type: types.BundleType("gpu-operator"), Success: true},
				{Type: types.BundleType("network-operator"), Success: false},
				{Type: types.BundleType("custom"), Success: true},
				{Type: types.BundleType("custom2"), Success: false},
			},
			want: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := &Output{
				Results: tt.results,
			}

			got := output.FailureCount()
			if got != tt.want {
				t.Errorf("FailureCount() = %d, want %d", got, tt.want)
			}
		})
	}
}

// TestOutput_formatBytes tests byte formatting
func TestOutput_formatBytes(t *testing.T) {
	tests := []struct {
		name  string
		bytes int64
		want  string
	}{
		{
			name:  "zero bytes",
			bytes: 0,
			want:  "0 B",
		},
		{
			name:  "bytes only",
			bytes: 512,
			want:  "512 B",
		},
		{
			name:  "exactly 1 KB",
			bytes: 1024,
			want:  "1.0 KB",
		},
		{
			name:  "KB range",
			bytes: 5120,
			want:  "5.0 KB",
		},
		{
			name:  "fractional KB",
			bytes: 1536,
			want:  "1.5 KB",
		},
		{
			name:  "exactly 1 MB",
			bytes: 1024 * 1024,
			want:  "1.0 MB",
		},
		{
			name:  "MB range",
			bytes: 5 * 1024 * 1024,
			want:  "5.0 MB",
		},
		{
			name:  "fractional MB",
			bytes: 1536 * 1024,
			want:  "1.5 MB",
		},
		{
			name:  "exactly 1 GB",
			bytes: 1024 * 1024 * 1024,
			want:  "1.0 GB",
		},
		{
			name:  "GB range",
			bytes: 3 * 1024 * 1024 * 1024,
			want:  "3.0 GB",
		},
		{
			name:  "fractional GB",
			bytes: int64(2.5 * 1024 * 1024 * 1024),
			want:  "2.5 GB",
		},
		{
			name:  "exactly 1 TB",
			bytes: 1024 * 1024 * 1024 * 1024,
			want:  "1.0 TB",
		},
		{
			name:  "TB range",
			bytes: 2 * 1024 * 1024 * 1024 * 1024,
			want:  "2.0 TB",
		},
		{
			name:  "exactly 1 PB",
			bytes: 1024 * 1024 * 1024 * 1024 * 1024,
			want:  "1.0 PB",
		},
		{
			name:  "PB range",
			bytes: 3 * 1024 * 1024 * 1024 * 1024 * 1024,
			want:  "3.0 PB",
		},
		{
			name:  "exactly 1 EB",
			bytes: 1024 * 1024 * 1024 * 1024 * 1024 * 1024,
			want:  "1.0 EB",
		},
		{
			name:  "EB range",
			bytes: 5 * 1024 * 1024 * 1024 * 1024 * 1024 * 1024,
			want:  "5.0 EB",
		},
		{
			name:  "maximum value",
			bytes: 9223372036854775807, // max int64
			want:  "8.0 EB",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatBytes(tt.bytes)
			if got != tt.want {
				t.Errorf("formatBytes(%d) = %s, want %s", tt.bytes, got, tt.want)
			}
		})
	}
}

// TestOutput_Summary tests summary generation
func TestOutput_Summary(t *testing.T) {
	tests := []struct {
		name         string
		output       *Output
		wantContains []string
	}{
		{
			name: "empty output",
			output: &Output{
				OutputDir:     "/output",
				Results:       []*Result{},
				TotalSize:     0,
				TotalFiles:    0,
				TotalDuration: 0,
			},
			wantContains: []string{
				"Generated 0 files",
				"0 B",
				"Success: 0/0",
			},
		},
		{
			name: "single successful bundler",
			output: &Output{
				OutputDir: "/output/bundles",
				Results: []*Result{
					{
						Type:    types.BundleType("gpu-operator"),
						Success: true,
						Files:   []string{"file1.yaml", "file2.yaml"},
						Size:    2048,
					},
				},
				TotalSize:     2048,
				TotalFiles:    2,
				TotalDuration: 5 * time.Second,
			},
			wantContains: []string{
				"Generated 2 files",
				"2.0 KB",
				"5s",
				"Success: 1/1",
			},
		},
		{
			name: "multiple bundlers with mixed results",
			output: &Output{
				OutputDir: "/output",
				Results: []*Result{
					{Type: types.BundleType("gpu-operator"), Success: true, Size: 1024},
					{Type: types.BundleType("network-operator"), Success: false, Size: 0},
					{Type: types.BundleType("custom"), Success: true, Size: 2048},
				},
				TotalSize:     3072,
				TotalFiles:    10,
				TotalDuration: 15 * time.Second,
			},
			wantContains: []string{
				"Generated 10 files",
				"3.0 KB",
				"15s",
				"Success: 2/3",
			},
		},
		{
			name: "large size formatting",
			output: &Output{
				OutputDir:     "/output",
				Results:       []*Result{{Type: types.BundleType("gpu-operator"), Success: true}},
				TotalSize:     5 * 1024 * 1024 * 1024, // 5 GB
				TotalFiles:    1000,
				TotalDuration: 2 * time.Minute,
			},
			wantContains: []string{
				"Generated 1000 files",
				"5.0 GB",
				"2m",
				"Success: 1/1",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.output.Summary()

			for _, want := range tt.wantContains {
				if !strings.Contains(got, want) {
					t.Errorf("Summary() missing expected content:\n  want: %s\n  got: %s", want, got)
				}
			}
		})
	}
}

// TestOutput_ByType tests result lookup by type
func TestOutput_ByType(t *testing.T) {
	result1 := &Result{Type: types.BundleType("gpu-operator"), Success: true}
	result2 := &Result{Type: types.BundleType("network-operator"), Success: false}
	result3 := &Result{Type: types.BundleType("custom"), Success: true}

	output := &Output{
		Results: []*Result{result1, result2, result3},
	}

	byType := output.ByType()

	if len(byType) != 3 {
		t.Errorf("ByType() returned %d results, want 3", len(byType))
	}

	// Verify each result is accessible by type
	if got := byType[types.BundleType("gpu-operator")]; got != result1 {
		t.Error("ByType()[gpu-operator] did not return correct result")
	}

	if got := byType[types.BundleType("network-operator")]; got != result2 {
		t.Error("ByType()[network-operator] did not return correct result")
	}

	if got := byType[types.BundleType("custom")]; got != result3 {
		t.Error("ByType()[custom] did not return correct result")
	}

	// Verify non-existent type returns nil
	if got := byType[types.BundleType("nonexistent")]; got != nil {
		t.Error("ByType()[nonexistent] should return nil")
	}
}

// TestOutput_ByType_Empty tests ByType with no results
func TestOutput_ByType_Empty(t *testing.T) {
	output := &Output{
		Results: []*Result{},
	}

	byType := output.ByType()

	if len(byType) != 0 {
		t.Errorf("ByType() for empty output returned %d results, want 0", len(byType))
	}
}

// TestOutput_FailedBundlers tests getting failed bundler types
func TestOutput_FailedBundlers(t *testing.T) {
	tests := []struct {
		name   string
		errors []BundleError
		want   []types.BundleType
	}{
		{
			name:   "no errors",
			errors: []BundleError{},
			want:   []types.BundleType{},
		},
		{
			name: "single error",
			errors: []BundleError{
				{BundlerType: types.BundleType("gpu-operator"), Error: "failed"},
			},
			want: []types.BundleType{
				types.BundleType("gpu-operator"),
			},
		},
		{
			name: "multiple errors",
			errors: []BundleError{
				{BundlerType: types.BundleType("gpu-operator"), Error: "error 1"},
				{BundlerType: types.BundleType("network-operator"), Error: "error 2"},
			},
			want: []types.BundleType{
				types.BundleType("gpu-operator"),
				types.BundleType("network-operator"),
			},
		},
		{
			name: "mixed with custom bundler",
			errors: []BundleError{
				{BundlerType: types.BundleType("network-operator"), Error: "error 1"},
				{BundlerType: types.BundleType("custom"), Error: "error 2"},
			},
			want: []types.BundleType{
				types.BundleType("network-operator"),
				types.BundleType("custom"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := &Output{
				Errors: tt.errors,
			}

			got := output.FailedBundlers()

			if len(got) != len(tt.want) {
				t.Errorf("FailedBundlers() returned %d types, want %d", len(got), len(tt.want))
				return
			}

			// Convert to map for easier comparison
			wantMap := make(map[types.BundleType]bool)
			for _, w := range tt.want {
				wantMap[w] = true
			}

			for _, g := range got {
				if !wantMap[g] {
					t.Errorf("FailedBundlers() returned unexpected type: %s", g)
				}
			}
		})
	}
}

// TestOutput_SuccessfulBundlers tests getting successful bundler types
func TestOutput_SuccessfulBundlers(t *testing.T) {
	tests := []struct {
		name    string
		results []*Result
		want    []types.BundleType
	}{
		{
			name:    "no results",
			results: []*Result{},
			want:    []types.BundleType{},
		},
		{
			name: "all successful",
			results: []*Result{
				{Type: types.BundleType("gpu-operator"), Success: true},
				{Type: types.BundleType("network-operator"), Success: true},
			},
			want: []types.BundleType{
				types.BundleType("gpu-operator"),
				types.BundleType("network-operator"),
			},
		},
		{
			name: "all failed",
			results: []*Result{
				{Type: types.BundleType("gpu-operator"), Success: false},
				{Type: types.BundleType("network-operator"), Success: false},
			},
			want: []types.BundleType{},
		},
		{
			name: "mixed results",
			results: []*Result{
				{Type: types.BundleType("gpu-operator"), Success: true},
				{Type: types.BundleType("network-operator"), Success: false},
				{Type: types.BundleType("custom1"), Success: true},
				{Type: types.BundleType("custom2"), Success: false},
			},
			want: []types.BundleType{
				types.BundleType("gpu-operator"),
				types.BundleType("custom1"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := &Output{
				Results: tt.results,
			}

			got := output.SuccessfulBundlers()

			if len(got) != len(tt.want) {
				t.Errorf("SuccessfulBundlers() returned %d types, want %d", len(got), len(tt.want))
				return
			}

			// Convert to map for easier comparison
			wantMap := make(map[types.BundleType]bool)
			for _, w := range tt.want {
				wantMap[w] = true
			}

			for _, g := range got {
				if !wantMap[g] {
					t.Errorf("SuccessfulBundlers() returned unexpected type: %s", g)
				}
			}
		})
	}
}

// TestBundleError tests BundleError struct
func TestBundleError(t *testing.T) {
	bundlerType := types.BundleType("gpu-operator")
	errorMsg := "failed to generate bundle"

	bundleErr := BundleError{
		BundlerType: bundlerType,
		Error:       errorMsg,
	}

	if bundleErr.BundlerType != bundlerType {
		t.Errorf("BundlerType = %s, want %s", bundleErr.BundlerType, bundlerType)
	}

	if bundleErr.Error != errorMsg {
		t.Errorf("Error = %s, want %s", bundleErr.Error, errorMsg)
	}
}

// TestOutput_CompleteWorkflow tests a complete output workflow
func TestOutput_CompleteWorkflow(t *testing.T) {
	// Create output with multiple results
	result1 := New(types.BundleType("gpu-operator"))
	result1.AddFile("/output/gpu/values.yaml", 1024)
	result1.AddFile("/output/gpu/manifest.yaml", 2048)
	result1.Duration = 3 * time.Second
	result1.MarkSuccess()

	result2 := New(types.BundleType("network-operator"))
	result2.AddFile("/output/network/values.yaml", 512)
	result2.AddError(errors.New("template error"))
	result2.Duration = 2 * time.Second
	// Note: not marking as success (failure case)

	result3 := New(types.BundleType("custom"))
	result3.AddFile("/output/custom/file1.yaml", 256)
	result3.AddFile("/output/custom/file2.yaml", 128)
	result3.Duration = 1 * time.Second
	result3.MarkSuccess()

	output := &Output{
		OutputDir:     "/output/bundles",
		Results:       []*Result{result1, result2, result3},
		TotalSize:     1024 + 2048 + 512 + 256 + 128,
		TotalFiles:    5,
		TotalDuration: 6 * time.Second,
		Errors: []BundleError{
			{BundlerType: types.BundleType("network-operator"), Error: "template error"},
		},
	}

	// Verify HasErrors
	if !output.HasErrors() {
		t.Error("Output should have errors")
	}

	// Verify SuccessCount
	if got := output.SuccessCount(); got != 2 {
		t.Errorf("SuccessCount() = %d, want 2", got)
	}

	// Verify FailureCount
	if got := output.FailureCount(); got != 1 {
		t.Errorf("FailureCount() = %d, want 1", got)
	}

	// Verify ByType
	byType := output.ByType()
	if len(byType) != 3 {
		t.Errorf("ByType() returned %d results, want 3", len(byType))
	}

	// Verify FailedBundlers
	failed := output.FailedBundlers()
	if len(failed) != 1 {
		t.Errorf("FailedBundlers() returned %d types, want 1", len(failed))
	}
	if failed[0] != types.BundleType("network-operator") {
		t.Errorf("FailedBundlers()[0] = %s, want %s", failed[0], types.BundleType("network-operator"))
	}

	// Verify SuccessfulBundlers
	successful := output.SuccessfulBundlers()
	if len(successful) != 2 {
		t.Errorf("SuccessfulBundlers() returned %d types, want 2", len(successful))
	}

	// Verify Summary contains expected information
	summary := output.Summary()
	expectedStrings := []string{
		"Generated 5 files",
		"6s",
		"Success: 2/3",
	}

	for _, expected := range expectedStrings {
		if !strings.Contains(summary, expected) {
			t.Errorf("Summary() missing: %s\nGot: %s", expected, summary)
		}
	}
}

// TestOutput_NilResults tests output with nil results slice
func TestOutput_NilResults(t *testing.T) {
	output := &Output{
		Results: nil,
	}

	if output.HasErrors() {
		t.Error("Output with nil results should not have errors")
	}

	if got := output.SuccessCount(); got != 0 {
		t.Errorf("SuccessCount() = %d, want 0", got)
	}

	if got := output.FailureCount(); got != 0 {
		t.Errorf("FailureCount() = %d, want 0", got)
	}

	byType := output.ByType()
	if len(byType) != 0 {
		t.Error("ByType() should return empty map for nil results")
	}

	failed := output.FailedBundlers()
	if len(failed) != 0 {
		t.Error("FailedBundlers() should return empty slice for nil results")
	}

	successful := output.SuccessfulBundlers()
	if len(successful) != 0 {
		t.Error("SuccessfulBundlers() should return empty slice for nil results")
	}
}
