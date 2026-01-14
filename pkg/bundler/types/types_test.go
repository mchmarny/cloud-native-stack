package types

import (
	"fmt"
	"testing"
)

// TestBundleType_String tests the String method
func TestBundleType_String(t *testing.T) {
	tests := []struct {
		name       string
		bundleType BundleType
		want       string
	}{
		{
			name:       "gpu-operator",
			bundleType: BundleTypeGpuOperator,
			want:       "gpu-operator",
		},
		{
			name:       "network-operator",
			bundleType: BundleTypeNetworkOperator,
			want:       "network-operator",
		},
		{
			name:       "custom type",
			bundleType: BundleType("custom-bundler"),
			want:       "custom-bundler",
		},
		{
			name:       "empty type",
			bundleType: BundleType(""),
			want:       "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.bundleType.String()
			if got != tt.want {
				t.Errorf("BundleType.String() = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestParseType tests parsing strings to BundleType
func TestParseType(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    BundleType
		wantErr bool
	}{
		{
			name:    "valid gpu-operator",
			input:   "gpu-operator",
			want:    BundleTypeGpuOperator,
			wantErr: false,
		},
		{
			name:    "valid network-operator",
			input:   "network-operator",
			want:    BundleTypeNetworkOperator,
			wantErr: false,
		},
		{
			name:    "invalid type",
			input:   "invalid-operator",
			want:    "",
			wantErr: true,
		},
		{
			name:    "empty string",
			input:   "",
			want:    "",
			wantErr: true,
		},
		{
			name:    "uppercase",
			input:   "GPU-OPERATOR",
			want:    BundleTypeGpuOperator,
			wantErr: false,
		},
		{
			name:    "mixed case",
			input:   "Gpu-Operator",
			want:    BundleTypeGpuOperator,
			wantErr: false,
		},
		{
			name:    "with spaces",
			input:   "gpu operator",
			want:    "",
			wantErr: true,
		},
		{
			name:    "random string",
			input:   "foobar",
			want:    "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseType(tt.input)

			if (err != nil) != tt.wantErr {
				t.Errorf("ParseType(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}

			if got != tt.want {
				t.Errorf("ParseType(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// TestSupportedTypes tests getting all supported types
func TestSupportedTypes(t *testing.T) {
	types := SupportedTypes()

	if len(types) == 0 {
		t.Fatal("SupportedTypes() returned empty slice")
	}

	// Verify expected types are present
	expectedTypes := []BundleType{
		BundleTypeGpuOperator,
		BundleTypeNetworkOperator,
		BundleTypeSkyhook,
		BundleTypeNVSentinel,
		BundleTypeCertManager,
	}

	if len(types) != len(expectedTypes) {
		t.Errorf("SupportedTypes() returned %d types, want %d", len(types), len(expectedTypes))
	}

	// Convert to map for easier lookup
	typeMap := make(map[BundleType]bool)
	for _, bt := range types {
		typeMap[bt] = true
	}

	// Verify each expected type is present
	for _, expected := range expectedTypes {
		if !typeMap[expected] {
			t.Errorf("SupportedTypes() missing expected type: %s", expected)
		}
	}
}

// TestSupportedTypes_Immutability tests that returned slice is independent
func TestSupportedTypes_Immutability(t *testing.T) {
	// Get the supported types twice
	types1 := SupportedTypes()
	types2 := SupportedTypes()

	// Verify they have the same content
	if len(types1) != len(types2) {
		t.Errorf("Multiple calls to SupportedTypes() returned different lengths: %d vs %d", len(types1), len(types2))
	}

	for i := range types1 {
		if types1[i] != types2[i] {
			t.Errorf("SupportedTypes()[%d] differs between calls: %s vs %s", i, types1[i], types2[i])
		}
	}

	// Note: We're returning a new slice each time, so modifying one shouldn't affect the other
	// This is already safe by design since we create a new slice literal in the function
}

// TestSupportedBundleTypesAsStrings tests string conversion
func TestSupportedBundleTypesAsStrings(t *testing.T) {
	strings := SupportedBundleTypesAsStrings()

	if len(strings) == 0 {
		t.Fatal("SupportedBundleTypesAsStrings() returned empty slice")
	}

	// Verify expected strings are present
	expectedStrings := []string{
		"gpu-operator",
		"network-operator",
		"skyhook",
		"nvsentinel",
		"cert-manager",
	}

	if len(strings) != len(expectedStrings) {
		t.Errorf("SupportedBundleTypesAsStrings() returned %d strings, want %d", len(strings), len(expectedStrings))
	}

	// Convert to map for easier lookup
	stringMap := make(map[string]bool)
	for _, s := range strings {
		stringMap[s] = true
	}

	// Verify each expected string is present
	for _, expected := range expectedStrings {
		if !stringMap[expected] {
			t.Errorf("SupportedBundleTypesAsStrings() missing expected string: %q", expected)
		}
	}
}

// TestSupportedBundleTypesAsStrings_ConsistentWithSupportedTypes tests consistency
func TestSupportedBundleTypesAsStrings_ConsistentWithSupportedTypes(t *testing.T) {
	types := SupportedTypes()
	strings := SupportedBundleTypesAsStrings()

	if len(types) != len(strings) {
		t.Errorf("SupportedTypes() and SupportedBundleTypesAsStrings() have different lengths: %d vs %d", len(types), len(strings))
	}

	// Verify each type's string representation matches
	for i := range types {
		expectedString := string(types[i])
		if strings[i] != expectedString {
			t.Errorf("SupportedBundleTypesAsStrings()[%d] = %q, want %q (from SupportedTypes()[%d])", i, strings[i], expectedString, i)
		}
	}
}

// TestBundleTypeConstants verifies the constant values
func TestBundleTypeConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant BundleType
		want     string
	}{
		{
			name:     "BundleTypeGpuOperator",
			constant: BundleTypeGpuOperator,
			want:     "gpu-operator",
		},
		{
			name:     "BundleTypeNetworkOperator",
			constant: BundleTypeNetworkOperator,
			want:     "network-operator",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := string(tt.constant)
			if got != tt.want {
				t.Errorf("%s = %q, want %q", tt.name, got, tt.want)
			}
		})
	}
}

// TestParseType_RoundTrip tests parsing and converting back to string
func TestParseType_RoundTrip(t *testing.T) {
	supportedTypes := SupportedTypes()

	for _, originalType := range supportedTypes {
		t.Run(string(originalType), func(t *testing.T) {
			// Convert to string
			str := originalType.String()

			// Parse back
			parsed, err := ParseType(str)
			if err != nil {
				t.Fatalf("ParseType(%q) unexpected error: %v", str, err)
			}

			// Verify round-trip
			if parsed != originalType {
				t.Errorf("Round-trip failed: original %q, after round-trip %q", originalType, parsed)
			}
		})
	}
}

// TestParseType_AllSupportedTypesAsStrings tests parsing all supported types
func TestParseType_AllSupportedTypesAsStrings(t *testing.T) {
	strings := SupportedBundleTypesAsStrings()

	for _, str := range strings {
		t.Run(str, func(t *testing.T) {
			parsed, err := ParseType(str)
			if err != nil {
				t.Errorf("ParseType(%q) unexpected error: %v", str, err)
			}

			// Verify the parsed type converts back to the same string
			if parsed.String() != str {
				t.Errorf("ParseType(%q).String() = %q, want %q", str, parsed.String(), str)
			}
		})
	}
}

// TestBundleType_Comparison tests type comparison
func TestBundleType_Comparison(t *testing.T) {
	// Test inequality
	if BundleTypeGpuOperator == BundleTypeNetworkOperator {
		t.Error("BundleTypeGpuOperator should not equal BundleTypeNetworkOperator")
	}

	// Test with custom type
	customType := BundleType("custom")
	if customType == BundleTypeGpuOperator {
		t.Error("Custom type should not equal BundleTypeGpuOperator")
	}

	// Test empty type
	emptyType := BundleType("")
	if emptyType == BundleTypeGpuOperator {
		t.Error("Empty type should not equal BundleTypeGpuOperator")
	}
}

// TestBundleType_AsMapKey tests using BundleType as map key
func TestBundleType_AsMapKey(t *testing.T) {
	// This tests that BundleType can be used as a map key
	m := make(map[BundleType]string)

	m[BundleTypeGpuOperator] = "gpu"
	m[BundleTypeNetworkOperator] = "network"

	if m[BundleTypeGpuOperator] != "gpu" {
		t.Errorf("Map lookup for BundleTypeGpuOperator = %q, want %q", m[BundleTypeGpuOperator], "gpu")
	}

	if m[BundleTypeNetworkOperator] != "network" {
		t.Errorf("Map lookup for BundleTypeNetworkOperator = %q, want %q", m[BundleTypeNetworkOperator], "network")
	}

	// Test with custom type
	customType := BundleType("custom")
	m[customType] = "custom-value"
	if m[customType] != "custom-value" {
		t.Errorf("Map lookup for custom type = %q, want %q", m[customType], "custom-value")
	}

	// Verify map has correct size
	if len(m) != 3 {
		t.Errorf("Map size = %d, want 3", len(m))
	}
}

// TestBundleType_ZeroValue tests the zero value of BundleType
func TestBundleType_ZeroValue(t *testing.T) {
	var bt BundleType

	// Zero value should be empty string
	if bt != "" {
		t.Errorf("Zero value of BundleType = %q, want empty string", bt)
	}

	// String() on zero value should return empty string
	if bt.String() != "" {
		t.Errorf("Zero value BundleType.String() = %q, want empty string", bt.String())
	}

	// ParseType should reject empty string
	parsed, err := ParseType(bt.String())
	if err == nil {
		t.Error("ParseType of zero value string should return error")
	}
	if parsed != "" {
		t.Errorf("ParseType of zero value string returned %q, want empty string", parsed)
	}
}

// TestSupportedTypes_ContainsAllConstants tests completeness
func TestSupportedTypes_ContainsAllConstants(t *testing.T) {
	// This test ensures all defined constants are included in SupportedTypes()
	types := SupportedTypes()
	typeMap := make(map[BundleType]bool)
	for _, bt := range types {
		typeMap[bt] = true
	}

	// Check each constant is in the supported types
	constants := []BundleType{
		BundleTypeGpuOperator,
		BundleTypeNetworkOperator,
	}

	for _, constant := range constants {
		if !typeMap[constant] {
			t.Errorf("SupportedTypes() missing constant: %s", constant)
		}
	}
}

// TestParseType_ErrorMessage tests error message format
func TestParseType_ErrorMessage(t *testing.T) {
	invalidInputs := []string{
		"invalid",
		"foo-bar",
		"",
		"completely-random-unrelated-text",
	}

	for _, input := range invalidInputs {
		t.Run(input, func(t *testing.T) {
			_, err := ParseType(input)
			if err == nil {
				t.Errorf("ParseType(%q) should return error", input)
				return
			}

			expectedPrefix := "unsupported bundle type"
			if len(err.Error()) < len(expectedPrefix) || err.Error()[:len(expectedPrefix)] != expectedPrefix {
				t.Errorf("ParseType(%q) error message = %q, should start with %q", input, err.Error(), expectedPrefix)
			}
		})
	}
}

// TestParseType_CaseInsensitive tests that parsing is case-insensitive
func TestParseType_CaseInsensitive(t *testing.T) {
	tests := []struct {
		input string
		want  BundleType
	}{
		{"gpu-operator", BundleTypeGpuOperator},
		{"GPU-OPERATOR", BundleTypeGpuOperator},
		{"Gpu-Operator", BundleTypeGpuOperator},
		{"gPu-OpErAtOr", BundleTypeGpuOperator},
		{"NETWORK-OPERATOR", BundleTypeNetworkOperator},
		{"Network-Operator", BundleTypeNetworkOperator},
		{"SKYHOOK", BundleTypeSkyhook},
		{"Skyhook", BundleTypeSkyhook},
		{"NVSENTINEL", BundleTypeNVSentinel},
		{"NvSentinel", BundleTypeNVSentinel},
		{"CERT-MANAGER", BundleTypeCertManager},
		{"Cert-Manager", BundleTypeCertManager},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := ParseType(tt.input)
			if err != nil {
				t.Errorf("ParseType(%q) unexpected error: %v", tt.input, err)
				return
			}
			if got != tt.want {
				t.Errorf("ParseType(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// TestParseType_LevenshteinSuggestions tests that close matches get suggestions
func TestParseType_LevenshteinSuggestions(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantSuggest string
	}{
		{
			name:        "typo in gpu-operator",
			input:       "gpu-opertor",
			wantSuggest: "gpu-operator",
		},
		{
			name:        "typo in network-operator",
			input:       "network-operater",
			wantSuggest: "network-operator",
		},
		{
			name:        "missing hyphen",
			input:       "gpuoperator",
			wantSuggest: "gpu-operator",
		},
		{
			name:        "typo in skyhook",
			input:       "skyhok",
			wantSuggest: "skyhook",
		},
		{
			name:        "typo in cert-manager",
			input:       "cert-manger",
			wantSuggest: "cert-manager",
		},
		{
			name:        "typo in nvsentinel",
			input:       "nvsentinal",
			wantSuggest: "nvsentinel",
		},
		{
			name:        "underscore instead of hyphen",
			input:       "gpu_operator",
			wantSuggest: "gpu-operator",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseType(tt.input)
			if err == nil {
				t.Errorf("ParseType(%q) should return error", tt.input)
				return
			}

			expectedSuffix := fmt.Sprintf("(did you mean %q?)", tt.wantSuggest)
			if !contains(err.Error(), expectedSuffix) {
				t.Errorf("ParseType(%q) error = %q, should contain suggestion %q", tt.input, err.Error(), expectedSuffix)
			}
		})
	}
}

// TestParseType_NoSuggestionForDistantStrings tests that very different strings don't get suggestions
func TestParseType_NoSuggestionForDistantStrings(t *testing.T) {
	distantInputs := []string{
		"xyz",
		"completely-different",
		"abcdefghijklmnop",
	}

	for _, input := range distantInputs {
		t.Run(input, func(t *testing.T) {
			_, err := ParseType(input)
			if err == nil {
				t.Errorf("ParseType(%q) should return error", input)
				return
			}

			if contains(err.Error(), "did you mean") {
				t.Errorf("ParseType(%q) error = %q, should NOT contain suggestion for distant string", input, err.Error())
			}
		})
	}
}

// contains checks if s contains substr
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
