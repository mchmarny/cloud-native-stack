package header

import (
	"testing"
	"time"
)

// Test API version constant - matches cns.nvidia.com/v1alpha1 used by snapshotter and recipe packages
const testAPIVersion = "cns.nvidia.com/v1alpha1"

func TestKind_String(t *testing.T) {
	tests := []struct {
		name string
		kind Kind
		want string
	}{
		{
			name: "Snapshot kind",
			kind: KindSnapshot,
			want: "Snapshot",
		},
		{
			name: "Recipe kind",
			kind: KindRecipe,
			want: "Recipe",
		},
		{
			name: "Custom kind",
			kind: Kind("CustomKind"),
			want: "CustomKind",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.kind.String(); got != tt.want {
				t.Errorf("Kind.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestKind_IsValid(t *testing.T) {
	tests := []struct {
		name string
		kind Kind
		want bool
	}{
		{
			name: "Snapshot is valid",
			kind: KindSnapshot,
			want: true,
		},
		{
			name: "Recipe is valid",
			kind: KindRecipe,
			want: true,
		},
		{
			name: "Empty kind is invalid",
			kind: Kind(""),
			want: false,
		},
		{
			name: "Unknown kind is invalid",
			kind: Kind("InvalidKind"),
			want: false,
		},
		{
			name: "Case sensitive - lowercase is invalid",
			kind: Kind("recipe"),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.kind.IsValid(); got != tt.want {
				t.Errorf("Kind.IsValid() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestWithMetadata(t *testing.T) {
	tests := []struct {
		name     string
		key      string
		value    string
		existing map[string]string
		want     map[string]string
	}{
		{
			name:     "Add metadata to empty header",
			key:      "test-key",
			value:    "test-value",
			existing: nil,
			want:     map[string]string{"test-key": "test-value"},
		},
		{
			name:     "Add metadata to existing metadata",
			key:      "new-key",
			value:    "new-value",
			existing: map[string]string{"existing-key": "existing-value"},
			want:     map[string]string{"existing-key": "existing-value", "new-key": "new-value"},
		},
		{
			name:     "Overwrite existing key",
			key:      "test-key",
			value:    "updated-value",
			existing: map[string]string{"test-key": "old-value"},
			want:     map[string]string{"test-key": "updated-value"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &Header{Metadata: tt.existing}
			opt := WithMetadata(tt.key, tt.value)
			opt(h)

			if len(h.Metadata) != len(tt.want) {
				t.Errorf("Metadata length = %v, want %v", len(h.Metadata), len(tt.want))
			}

			for key, wantValue := range tt.want {
				if gotValue, exists := h.Metadata[key]; !exists {
					t.Errorf("Expected key %q not found in metadata", key)
				} else if gotValue != wantValue {
					t.Errorf("Metadata[%q] = %v, want %v", key, gotValue, wantValue)
				}
			}
		})
	}
}

func TestWithKind(t *testing.T) {
	tests := []struct {
		name string
		kind Kind
		want Kind
	}{
		{
			name: "Set Snapshot kind",
			kind: KindSnapshot,
			want: KindSnapshot,
		},
		{
			name: "Set Recipe kind",
			kind: KindRecipe,
			want: KindRecipe,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &Header{}
			opt := WithKind(tt.kind)
			opt(h)

			if h.Kind != tt.want {
				t.Errorf("Kind = %v, want %v", h.Kind, tt.want)
			}
		})
	}
}

func TestWithAPIVersion(t *testing.T) {
	tests := []struct {
		name    string
		version string
		want    string
	}{
		{
			name:    "Set v1alpha1 API version",
			version: "cns.nvidia.com/v1alpha1",
			want:    "cns.nvidia.com/v1alpha1",
		},
		{
			name:    "Set custom API version",
			version: "custom.example.com/v2",
			want:    "custom.example.com/v2",
		},
		{
			name:    "Set empty API version",
			version: "",
			want:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &Header{}
			opt := WithAPIVersion(tt.version)
			opt(h)

			if h.APIVersion != tt.want {
				t.Errorf("APIVersion = %v, want %v", h.APIVersion, tt.want)
			}
		})
	}
}

func TestHeader_SetKind(t *testing.T) {
	tests := []struct {
		name string
		kind Kind
		want Kind
	}{
		{
			name: "Set Snapshot kind",
			kind: KindSnapshot,
			want: KindSnapshot,
		},
		{
			name: "Set Recipe kind",
			kind: KindRecipe,
			want: KindRecipe,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &Header{Kind: KindSnapshot}
			h.SetKind(tt.kind)

			if h.Kind != tt.want {
				t.Errorf("Kind = %v, want %v", h.Kind, tt.want)
			}
		})
	}
}

func TestNew(t *testing.T) {
	tests := []struct {
		name  string
		opts  []Option
		check func(*testing.T, *Header)
	}{
		{
			name: "Create header with no options",
			opts: nil,
			check: func(t *testing.T, h *Header) {
				if h.Metadata == nil {
					t.Error("Metadata should be initialized")
				}
				if len(h.Metadata) != 0 {
					t.Errorf("Metadata should be empty, got %d items", len(h.Metadata))
				}
			},
		},
		{
			name: "Create header with kind",
			opts: []Option{WithKind(KindSnapshot)},
			check: func(t *testing.T, h *Header) {
				if h.Kind != KindSnapshot {
					t.Errorf("Kind = %v, want %v", h.Kind, KindSnapshot)
				}
			},
		},
		{
			name: "Create header with API version",
			opts: []Option{WithAPIVersion("test.example.com/v1")},
			check: func(t *testing.T, h *Header) {
				if h.APIVersion != "test.example.com/v1" {
					t.Errorf("APIVersion = %v, want %v", h.APIVersion, "test.example.com/v1")
				}
			},
		},
		{
			name: "Create header with metadata",
			opts: []Option{WithMetadata("key1", "value1"), WithMetadata("key2", "value2")},
			check: func(t *testing.T, h *Header) {
				if len(h.Metadata) != 2 {
					t.Errorf("Metadata length = %v, want 2", len(h.Metadata))
				}
				if h.Metadata["key1"] != "value1" {
					t.Errorf("Metadata[key1] = %v, want value1", h.Metadata["key1"])
				}
				if h.Metadata["key2"] != "value2" {
					t.Errorf("Metadata[key2] = %v, want value2", h.Metadata["key2"])
				}
			},
		},
		{
			name: "Create header with all options",
			opts: []Option{
				WithKind(KindRecipe),
				WithAPIVersion("cns.nvidia.com/v1alpha1"),
				WithMetadata("version", "1.0.0"),
				WithMetadata("created", "2025-01-01T00:00:00Z"),
			},
			check: func(t *testing.T, h *Header) {
				if h.Kind != KindRecipe {
					t.Errorf("Kind = %v, want %v", h.Kind, KindRecipe)
				}
				if h.APIVersion != "cns.nvidia.com/v1alpha1" {
					t.Errorf("APIVersion = %v, want cns.nvidia.com/v1alpha1", h.APIVersion)
				}
				if len(h.Metadata) != 2 {
					t.Errorf("Metadata length = %v, want 2", len(h.Metadata))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := New(tt.opts...)
			if h == nil {
				t.Fatal("New() returned nil")
			}
			tt.check(t, h)
		})
	}
}

func TestHeader_Init(t *testing.T) {
	tests := []struct {
		name    string
		kind    Kind
		version string
		check   func(*testing.T, *Header)
	}{
		{
			name:    "Init Snapshot with version",
			kind:    KindSnapshot,
			version: "v1.0.0",
			check: func(t *testing.T, h *Header) {
				if h.Kind != KindSnapshot {
					t.Errorf("Kind = %v, want %v", h.Kind, KindSnapshot)
				}
				if h.APIVersion != testAPIVersion {
					t.Errorf("APIVersion = %v, want %s", h.APIVersion, testAPIVersion)
				}
				if h.Metadata == nil {
					t.Fatal("Metadata is nil")
				}
				if _, exists := h.Metadata["timestamp"]; !exists {
					t.Error("timestamp not found in metadata")
				}
				if v := h.Metadata["version"]; v != "v1.0.0" {
					t.Errorf("version = %v, want v1.0.0", v)
				}
			},
		},
		{
			name:    "Init Recipe without version",
			kind:    KindRecipe,
			version: "",
			check: func(t *testing.T, h *Header) {
				if h.Kind != KindRecipe {
					t.Errorf("Kind = %v, want %v", h.Kind, KindRecipe)
				}
				if h.APIVersion != testAPIVersion {
					t.Errorf("APIVersion = %v, want %s", h.APIVersion, testAPIVersion)
				}
				if _, exists := h.Metadata["recipe-timestamp"]; !exists {
					t.Error("recipe-timestamp not found in metadata")
				}
				if _, exists := h.Metadata["recipe-version"]; exists {
					t.Error("recipe-version should not exist when version is empty")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &Header{}
			h.Init(tt.kind, testAPIVersion, tt.version)
			tt.check(t, h)
		})
	}
}

func TestHeader_Init_TimestampFormat(t *testing.T) {
	h := &Header{}
	h.Init(KindSnapshot, testAPIVersion, "v1.0.0")

	timestamp, exists := h.Metadata["timestamp"]
	if !exists {
		t.Fatal("timestamp not found in metadata")
	}

	// Parse the timestamp to ensure it's valid RFC3339
	parsedTime, err := time.Parse(time.RFC3339, timestamp)
	if err != nil {
		t.Errorf("Failed to parse timestamp as RFC3339: %v", err)
	}

	// Verify timestamp is recent (within last minute)
	now := time.Now().UTC()
	diff := now.Sub(parsedTime)
	if diff < 0 || diff > time.Minute {
		t.Errorf("Timestamp %v is not recent (diff: %v)", timestamp, diff)
	}
}

func TestHeader_Init_OverwritesExistingData(t *testing.T) {
	h := &Header{
		Kind:       KindRecipe,
		APIVersion: "old.example.com/v1",
		Metadata: map[string]string{
			"existing-key": "existing-value",
		},
	}

	h.Init(KindSnapshot, testAPIVersion, "v2.0.0")

	// Check that old data is replaced
	if h.Kind != KindSnapshot {
		t.Errorf("Kind was not updated, got %v, want %v", h.Kind, KindSnapshot)
	}

	if h.APIVersion != testAPIVersion {
		t.Errorf("APIVersion was not updated, got %v, want %s", h.APIVersion, testAPIVersion)
	}

	// Metadata should be completely replaced
	if _, exists := h.Metadata["existing-key"]; exists {
		t.Error("Old metadata key should have been removed")
	}

	if _, exists := h.Metadata["version"]; !exists {
		t.Error("New metadata should be present")
	}
}

func TestConstants(t *testing.T) {
	// Verify constant values haven't changed
	if KindSnapshot != "Snapshot" {
		t.Errorf("KindSnapshot = %v, want Snapshot", KindSnapshot)
	}
	if KindRecipe != "Recipe" {
		t.Errorf("KindRecipe = %v, want Recipe", KindRecipe)
	}
	// Note: API version constants moved to resource-specific packages
	// - snapshotter.FullAPIVersion for Snapshot resources
	// - recipe.FullAPIVersion for Recipe resources
	// This allows independent evolution of each resource type's API version
}
