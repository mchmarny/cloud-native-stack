package recipe

import (
	"os"
	"path/filepath"
	"testing"

	"gopkg.in/yaml.v3"
)

// testEmptyRegistryContent is a minimal registry.yaml for testing.
const testEmptyRegistryContent = `apiVersion: cns.nvidia.com/v1alpha1
kind: ComponentRegistry
components: []
`

// TestEmbeddedDataProvider tests the embedded data provider.
func TestEmbeddedDataProvider(t *testing.T) {
	provider := NewEmbeddedDataProvider(dataFS, "data")

	t.Run("read existing file", func(t *testing.T) {
		data, err := provider.ReadFile("registry.yaml")
		if err != nil {
			t.Fatalf("failed to read registry.yaml: %v", err)
		}
		if len(data) == 0 {
			t.Error("registry.yaml is empty")
		}
	})

	t.Run("read non-existent file", func(t *testing.T) {
		_, err := provider.ReadFile("non-existent.yaml")
		if err == nil {
			t.Error("expected error for non-existent file")
		}
	})

	t.Run("source returns embedded", func(t *testing.T) {
		source := provider.Source("registry.yaml")
		if source != sourceEmbedded {
			t.Errorf("expected source %q, got %q", sourceEmbedded, source)
		}
	})
}

// TestLayeredDataProvider_RequiresRegistry tests that external dir must have registry.yaml.
func TestLayeredDataProvider_RequiresRegistry(t *testing.T) {
	// Create temp directory without registry.yaml
	tmpDir := t.TempDir()

	embedded := NewEmbeddedDataProvider(dataFS, "data")
	_, err := NewLayeredDataProvider(embedded, LayeredProviderConfig{
		ExternalDir: tmpDir,
	})

	if err == nil {
		t.Error("expected error when registry.yaml is missing")
	}
}

// TestLayeredDataProvider_MergesRegistry tests registry merging.
func TestLayeredDataProvider_MergesRegistry(t *testing.T) {
	// Create temp directory with registry.yaml
	tmpDir := t.TempDir()

	// Create a registry with a custom component
	registryContent := `apiVersion: cns.nvidia.com/v1alpha1
kind: ComponentRegistry
components:
  - name: custom-component
    displayName: Custom Component
    helm:
      defaultRepository: https://example.com/charts
      defaultChart: custom/custom-component
`
	if err := os.WriteFile(filepath.Join(tmpDir, "registry.yaml"), []byte(registryContent), 0600); err != nil {
		t.Fatalf("failed to write registry.yaml: %v", err)
	}

	embedded := NewEmbeddedDataProvider(dataFS, "data")
	provider, err := NewLayeredDataProvider(embedded, LayeredProviderConfig{
		ExternalDir: tmpDir,
	})
	if err != nil {
		t.Fatalf("failed to create layered provider: %v", err)
	}

	// Read merged registry
	data, err := provider.ReadFile("registry.yaml")
	if err != nil {
		t.Fatalf("failed to read registry.yaml: %v", err)
	}

	// Should contain both embedded and custom components
	content := string(data)
	if !contains(content, "custom-component") {
		t.Error("merged registry should contain custom-component from external")
	}
	if !contains(content, "gpu-operator") {
		t.Error("merged registry should contain gpu-operator from embedded")
	}
}

// TestLayeredDataProvider_OverridesFile tests file replacement.
func TestLayeredDataProvider_OverridesFile(t *testing.T) {
	tmpDir := t.TempDir()

	// Create registry.yaml (required)
	registryContent := testEmptyRegistryContent
	if err := os.WriteFile(filepath.Join(tmpDir, "registry.yaml"), []byte(registryContent), 0600); err != nil {
		t.Fatalf("failed to write registry.yaml: %v", err)
	}

	// Create a custom base.yaml that will override embedded
	baseContent := `apiVersion: cns.nvidia.com/v1alpha1
kind: RecipeMetadata
metadata:
  name: custom-base
spec:
  components: []
`
	if err := os.WriteFile(filepath.Join(tmpDir, "base.yaml"), []byte(baseContent), 0600); err != nil {
		t.Fatalf("failed to write base.yaml: %v", err)
	}

	embedded := NewEmbeddedDataProvider(dataFS, "data")
	provider, err := NewLayeredDataProvider(embedded, LayeredProviderConfig{
		ExternalDir: tmpDir,
	})
	if err != nil {
		t.Fatalf("failed to create layered provider: %v", err)
	}

	// Read base.yaml - should get external version
	data, err := provider.ReadFile("base.yaml")
	if err != nil {
		t.Fatalf("failed to read base.yaml: %v", err)
	}

	content := string(data)
	if !contains(content, "custom-base") {
		t.Error("base.yaml should be from external directory")
	}

	// Check source
	source := provider.Source("base.yaml")
	if source != "external" {
		t.Errorf("expected source 'external', got %q", source)
	}
}

// TestLayeredDataProvider_AddsNewFile tests adding new files.
func TestLayeredDataProvider_AddsNewFile(t *testing.T) {
	tmpDir := t.TempDir()

	// Create registry.yaml (required)
	registryContent := testEmptyRegistryContent
	if err := os.WriteFile(filepath.Join(tmpDir, "registry.yaml"), []byte(registryContent), 0600); err != nil {
		t.Fatalf("failed to write registry.yaml: %v", err)
	}

	// Create a new overlay that doesn't exist in embedded
	overlaysDir := filepath.Join(tmpDir, "overlays")
	if err := os.MkdirAll(overlaysDir, 0755); err != nil {
		t.Fatalf("failed to create overlays dir: %v", err)
	}

	overlayContent := `apiVersion: cns.nvidia.com/v1alpha1
kind: RecipeMetadata
metadata:
  name: custom-overlay
spec:
  criteria:
    service: custom
  components: []
`
	if err := os.WriteFile(filepath.Join(overlaysDir, "custom-overlay.yaml"), []byte(overlayContent), 0600); err != nil {
		t.Fatalf("failed to write custom-overlay.yaml: %v", err)
	}

	embedded := NewEmbeddedDataProvider(dataFS, "data")
	provider, err := NewLayeredDataProvider(embedded, LayeredProviderConfig{
		ExternalDir: tmpDir,
	})
	if err != nil {
		t.Fatalf("failed to create layered provider: %v", err)
	}

	// Read new overlay
	data, err := provider.ReadFile("overlays/custom-overlay.yaml")
	if err != nil {
		t.Fatalf("failed to read custom-overlay.yaml: %v", err)
	}

	content := string(data)
	if !contains(content, "custom-overlay") {
		t.Error("should be able to read custom overlay from external")
	}
}

// TestLayeredDataProvider_SecurityChecks tests security validations.
func TestLayeredDataProvider_SecurityChecks(t *testing.T) {
	t.Run("rejects symlinks", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create registry.yaml
		registryContent := testEmptyRegistryContent
		if err := os.WriteFile(filepath.Join(tmpDir, "registry.yaml"), []byte(registryContent), 0600); err != nil {
			t.Fatalf("failed to write registry.yaml: %v", err)
		}

		// Create a symlink
		symlinkPath := filepath.Join(tmpDir, "symlink.yaml")
		targetPath := filepath.Join(tmpDir, "registry.yaml")
		if err := os.Symlink(targetPath, symlinkPath); err != nil {
			t.Skipf("cannot create symlinks: %v", err)
		}

		embedded := NewEmbeddedDataProvider(dataFS, "data")
		_, err := NewLayeredDataProvider(embedded, LayeredProviderConfig{
			ExternalDir:   tmpDir,
			AllowSymlinks: false,
		})

		if err == nil {
			t.Error("expected error for symlink")
		}
	})

	t.Run("rejects large files", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create registry.yaml that exceeds size limit
		largeContent := make([]byte, 100) // Small for test, but we'll set a tiny limit
		if err := os.WriteFile(filepath.Join(tmpDir, "registry.yaml"), largeContent, 0600); err != nil {
			t.Fatalf("failed to write registry.yaml: %v", err)
		}

		embedded := NewEmbeddedDataProvider(dataFS, "data")
		_, err := NewLayeredDataProvider(embedded, LayeredProviderConfig{
			ExternalDir: tmpDir,
			MaxFileSize: 10, // Very small limit
		})

		if err == nil {
			t.Error("expected error for file exceeding size limit")
		}
	})

	t.Run("rejects missing directory", func(t *testing.T) {
		embedded := NewEmbeddedDataProvider(dataFS, "data")
		_, err := NewLayeredDataProvider(embedded, LayeredProviderConfig{
			ExternalDir: "/non/existent/path",
		})

		if err == nil {
			t.Error("expected error for non-existent directory")
		}
	})
}

// TestLayeredDataProvider_FallsBackToEmbedded tests fallback behavior.
func TestLayeredDataProvider_FallsBackToEmbedded(t *testing.T) {
	tmpDir := t.TempDir()

	// Create registry.yaml (required)
	registryContent := testEmptyRegistryContent
	if err := os.WriteFile(filepath.Join(tmpDir, "registry.yaml"), []byte(registryContent), 0600); err != nil {
		t.Fatalf("failed to write registry.yaml: %v", err)
	}

	embedded := NewEmbeddedDataProvider(dataFS, "data")
	provider, err := NewLayeredDataProvider(embedded, LayeredProviderConfig{
		ExternalDir: tmpDir,
	})
	if err != nil {
		t.Fatalf("failed to create layered provider: %v", err)
	}

	// Read base.yaml - should fall back to embedded since we didn't override it
	data, err := provider.ReadFile("base.yaml")
	if err != nil {
		t.Fatalf("failed to read base.yaml: %v", err)
	}

	if len(data) == 0 {
		t.Error("base.yaml should not be empty")
	}

	// Source should be embedded
	source := provider.Source("base.yaml")
	if source != "embedded" {
		t.Errorf("expected source 'embedded', got %q", source)
	}
}

// TestLayeredDataProvider_IntegrationWithRegistry tests that the layered provider
// correctly merges registry files by testing the merged content directly.
func TestLayeredDataProvider_IntegrationWithRegistry(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a registry with an additional custom component
	registryContent := `apiVersion: cns.nvidia.com/v1alpha1
kind: ComponentRegistry
components:
  - name: custom-operator
    displayName: Custom Operator
    helm:
      defaultRepository: https://custom.example.com/charts
      defaultChart: custom/custom-operator
      defaultVersion: v1.0.0
`
	if err := os.WriteFile(filepath.Join(tmpDir, "registry.yaml"), []byte(registryContent), 0600); err != nil {
		t.Fatalf("failed to write registry.yaml: %v", err)
	}

	// Create layered provider
	embedded := NewEmbeddedDataProvider(dataFS, "data")
	layered, err := NewLayeredDataProvider(embedded, LayeredProviderConfig{
		ExternalDir: tmpDir,
	})
	if err != nil {
		t.Fatalf("failed to create layered provider: %v", err)
	}

	// Read the merged registry directly from the provider
	mergedData, err := layered.ReadFile("registry.yaml")
	if err != nil {
		t.Fatalf("failed to read merged registry: %v", err)
	}

	// Parse the merged registry
	var registry ComponentRegistry
	if err := yaml.Unmarshal(mergedData, &registry); err != nil {
		t.Fatalf("failed to parse merged registry: %v", err)
	}

	// Build index for lookup
	registry.byName = make(map[string]*ComponentConfig, len(registry.Components))
	for i := range registry.Components {
		comp := &registry.Components[i]
		registry.byName[comp.Name] = comp
	}

	// Verify custom component exists
	customComp := registry.Get("custom-operator")
	if customComp == nil {
		t.Error("custom-operator should exist in merged registry")
	} else if customComp.DisplayName != "Custom Operator" {
		t.Errorf("custom-operator displayName = %q, want 'Custom Operator'", customComp.DisplayName)
	}

	// Verify embedded components still exist
	gpuOp := registry.Get("gpu-operator")
	if gpuOp == nil {
		t.Error("gpu-operator should still exist from embedded registry")
	}

	certManager := registry.Get("cert-manager")
	if certManager == nil {
		t.Error("cert-manager should still exist from embedded registry")
	}
}

// TestLayeredDataProvider_OverrideComponentValues tests overriding component values files.
func TestLayeredDataProvider_OverrideComponentValues(t *testing.T) {
	tmpDir := t.TempDir()

	// Create required registry.yaml
	if err := os.WriteFile(filepath.Join(tmpDir, "registry.yaml"), []byte(testEmptyRegistryContent), 0600); err != nil {
		t.Fatalf("failed to write registry.yaml: %v", err)
	}

	// Create custom values file for cert-manager
	componentsDir := filepath.Join(tmpDir, "components", "cert-manager")
	if err := os.MkdirAll(componentsDir, 0755); err != nil {
		t.Fatalf("failed to create components dir: %v", err)
	}

	customValues := `# Custom values for testing
installCRDs: false
customField: customValue
`
	if err := os.WriteFile(filepath.Join(componentsDir, "values.yaml"), []byte(customValues), 0600); err != nil {
		t.Fatalf("failed to write custom values: %v", err)
	}

	embedded := NewEmbeddedDataProvider(dataFS, "data")
	provider, err := NewLayeredDataProvider(embedded, LayeredProviderConfig{
		ExternalDir: tmpDir,
	})
	if err != nil {
		t.Fatalf("failed to create layered provider: %v", err)
	}

	// Read the custom values
	data, err := provider.ReadFile("components/cert-manager/values.yaml")
	if err != nil {
		t.Fatalf("failed to read custom values: %v", err)
	}

	content := string(data)
	if !contains(content, "customField") {
		t.Error("custom values should contain customField")
	}
	if !contains(content, "customValue") {
		t.Error("custom values should contain customValue")
	}

	// Verify source is external
	source := provider.Source("components/cert-manager/values.yaml")
	if source != "external" {
		t.Errorf("expected source 'external', got %q", source)
	}
}

// TestDataProviderGeneration tests that generation increments correctly.
func TestDataProviderGeneration(t *testing.T) {
	// Save original state
	originalProvider := globalDataProvider
	originalGen := dataProviderGeneration
	defer func() {
		globalDataProvider = originalProvider
		dataProviderGeneration = originalGen
	}()

	startGen := GetDataProviderGeneration()

	// Setting a provider should increment generation
	embedded := NewEmbeddedDataProvider(dataFS, "data")
	SetDataProvider(embedded)

	newGen := GetDataProviderGeneration()
	if newGen != startGen+1 {
		t.Errorf("expected generation %d, got %d", startGen+1, newGen)
	}

	// Setting again should increment again
	SetDataProvider(embedded)
	if GetDataProviderGeneration() != startGen+2 {
		t.Errorf("expected generation %d, got %d", startGen+2, GetDataProviderGeneration())
	}
}

// contains checks if s contains substr.
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr ||
		len(s) > 0 && len(substr) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
