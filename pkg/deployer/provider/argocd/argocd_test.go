package argocd

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/NVIDIA/cloud-native-stack/pkg/deployer/internal"
	"github.com/NVIDIA/cloud-native-stack/pkg/deployer/registry"
	"github.com/NVIDIA/cloud-native-stack/pkg/deployer/types"
	"github.com/NVIDIA/cloud-native-stack/pkg/recipe"
)

const testVersion = "v1.0.0"

// Verify Deployer implements registry.Deployer interface at compile time
var _ registry.Deployer = (*Deployer)(nil)

func TestNewDeployer(t *testing.T) {
	deployer := NewDeployer()

	if deployer == nil {
		t.Fatal("NewDeployer() returned nil")
	}
}

func TestDeployer_Generate(t *testing.T) {
	deployer := &Deployer{}
	ctx := context.Background()

	// Create test directory
	tmpDir, err := os.MkdirTemp("", "argocd-deployer-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create test recipe result
	recipeResult := &recipe.RecipeResult{}
	recipeResult.Metadata.Version = testVersion
	recipeResult.ComponentRefs = []recipe.ComponentRef{
		{
			Name:    "gpu-operator",
			Version: "v25.3.3",
			Source:  "https://helm.ngc.nvidia.com/nvidia",
		},
		{
			Name:    "network-operator",
			Version: "v25.4.0",
			Source:  "https://helm.ngc.nvidia.com/nvidia",
		},
	}

	// Call Generate
	artifacts, err := deployer.Generate(ctx, recipeResult, tmpDir)

	if err != nil {
		t.Fatalf("Generate() failed: %v", err)
	}

	if artifacts == nil {
		t.Fatal("Generate() returned nil artifacts")
	}

	// Verify success
	if !artifacts.Success {
		t.Errorf("Generate() artifacts.Success = false, want true (error: %s)", artifacts.Error)
	}

	// Verify Application manifests were created in component directories
	expectedFiles := []string{
		"gpu-operator/argocd/application.yaml",
		"network-operator/argocd/application.yaml",
		"app-of-apps.yaml",
		"README.md",
	}

	for _, expectedFile := range expectedFiles {
		fullPath := filepath.Join(tmpDir, expectedFile)
		if _, err := os.Stat(fullPath); os.IsNotExist(err) {
			t.Errorf("Generate() did not create %s", expectedFile)
		}
	}

	// Verify Files includes all expected files
	if len(artifacts.Files) < len(expectedFiles) {
		t.Errorf("Generate() returned %d files, want at least %d", len(artifacts.Files), len(expectedFiles))
	}

	// Verify README content
	if artifacts.ReadmeContent == "" {
		t.Error("Generate() did not set ReadmeContent")
	}

	// Verify Duration is set
	if artifacts.Duration == 0 {
		t.Error("Generate() did not set Duration")
	}
}

func TestDeployer_Generate_EmptyComponents(t *testing.T) {
	deployer := &Deployer{}
	ctx := context.Background()

	tmpDir, err := os.MkdirTemp("", "argocd-deployer-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create recipe with no components
	recipeResult := &recipe.RecipeResult{}
	recipeResult.Metadata.Version = testVersion
	recipeResult.ComponentRefs = []recipe.ComponentRef{}

	artifacts, err := deployer.Generate(ctx, recipeResult, tmpDir)

	if err != nil {
		t.Fatalf("Generate() with empty components failed: %v", err)
	}

	if !artifacts.Success {
		t.Errorf("Generate() with empty components failed: %s", artifacts.Error)
	}

	// Should still create app-of-apps.yaml at root
	appOfAppsPath := filepath.Join(tmpDir, "app-of-apps.yaml")
	if _, err := os.Stat(appOfAppsPath); os.IsNotExist(err) {
		t.Error("Generate() did not create app-of-apps.yaml for empty components")
	}
}

func TestDeployer_Generate_InvalidDirectory(t *testing.T) {
	deployer := &Deployer{}
	ctx := context.Background()

	// Use a non-existent directory
	nonExistentDir := "/nonexistent/path/that/does/not/exist"

	recipeResult := &recipe.RecipeResult{}
	recipeResult.Metadata.Version = testVersion
	recipeResult.ComponentRefs = []recipe.ComponentRef{
		{Name: "test", Version: testVersion},
	}

	artifacts, err := deployer.Generate(ctx, recipeResult, nonExistentDir)

	if err == nil {
		t.Error("Generate() with invalid directory did not return error")
	}

	if artifacts.Success {
		t.Error("Generate() with invalid directory has Success=true")
	}

	if artifacts.Error == "" {
		t.Error("Generate() with invalid directory did not set Error message")
	}
}

func TestInternalGetNamespaceForComponent(t *testing.T) {
	tests := []struct {
		componentName string
		want          string
	}{
		{"gpu-operator", "gpu-operator"},
		{"network-operator", "network-operator"},
		{"cert-manager", "cert-manager"},
		{"unknown-component", "default"},
		{"", "default"},
	}

	for _, tt := range tests {
		t.Run(tt.componentName, func(t *testing.T) {
			got := internal.GetNamespaceForComponent(tt.componentName)
			if got != tt.want {
				t.Errorf("GetNamespaceForComponent(%s) = %s, want %s", tt.componentName, got, tt.want)
			}
		})
	}
}

func TestApplicationData_Fields(t *testing.T) {
	data := ApplicationData{
		Name:         "test-app",
		Source:       "https://example.com",
		Version:      "v1.2.3",
		Namespace:    "test-ns",
		SyncWave:     0,
		ValuesFile:   "test-app/values.yaml",
		HasManifests: true,
		ManifestsDir: "test-app/manifests",
	}

	if data.Name != "test-app" {
		t.Errorf("ApplicationData.Name = %s, want test-app", data.Name)
	}

	if data.Source != "https://example.com" {
		t.Errorf("ApplicationData.Source = %s, want https://example.com", data.Source)
	}

	if data.Version != "v1.2.3" {
		t.Errorf("ApplicationData.Version = %s, want v1.2.3", data.Version)
	}

	if data.Namespace != "test-ns" {
		t.Errorf("ApplicationData.Namespace = %s, want test-ns", data.Namespace)
	}

	if data.ValuesFile != "test-app/values.yaml" {
		t.Errorf("ApplicationData.ValuesFile = %s, want test-app/values.yaml", data.ValuesFile)
	}

	if !data.HasManifests {
		t.Error("ApplicationData.HasManifests = false, want true")
	}

	if data.ManifestsDir != "test-app/manifests" {
		t.Errorf("ApplicationData.ManifestsDir = %s, want test-app/manifests", data.ManifestsDir)
	}
}

func TestInternalReadmeData_Fields(t *testing.T) {
	data := internal.ReadmeData{
		Timestamp: time.Now().Format(time.RFC3339),
		Components: []internal.ComponentInfo{
			{Name: "test-component", Version: "v1.2.3"},
		},
	}

	if data.Timestamp == "" {
		t.Error("ReadmeData.Timestamp is empty")
	}

	if len(data.Components) != 1 {
		t.Errorf("ReadmeData.Components length = %d, want 1", len(data.Components))
	}
}

func TestInternalComponentInfo_Fields(t *testing.T) {
	info := internal.ComponentInfo{
		Name:    "test-component",
		Version: "v1.2.3",
	}

	if info.Name != "test-component" {
		t.Errorf("ComponentInfo.Name = %s, want test-component", info.Name)
	}

	if info.Version != "v1.2.3" {
		t.Errorf("ComponentInfo.Version = %s, want v1.2.3", info.Version)
	}
}

func TestInternalRenderTemplate(t *testing.T) {
	tests := []struct {
		name     string
		template string
		data     interface{}
		want     string
		wantErr  bool
	}{
		{
			name:     "simple template",
			template: "Hello {{ .Name }}",
			data:     struct{ Name string }{Name: "World"},
			want:     "Hello World",
			wantErr:  false,
		},
		{
			name:     "invalid template syntax",
			template: "Hello {{ .Name",
			data:     struct{ Name string }{Name: "World"},
			want:     "",
			wantErr:  true,
		},
		{
			name:     "nil data",
			template: "Static content",
			data:     nil,
			want:     "Static content",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := internal.RenderTemplate(tt.template, tt.data)

			if (err != nil) != tt.wantErr {
				t.Errorf("renderTemplate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && got != tt.want {
				t.Errorf("renderTemplate() = %s, want %s", got, tt.want)
			}
		})
	}
}

func TestDeployer_Generate_VerifiesApplicationContent(t *testing.T) {
	deployer := &Deployer{}
	ctx := context.Background()

	tmpDir, err := os.MkdirTemp("", "argocd-deployer-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	recipeResult := &recipe.RecipeResult{}
	recipeResult.Metadata.Version = "v1.0.0"
	recipeResult.ComponentRefs = []recipe.ComponentRef{
		{
			Name:    "gpu-operator",
			Version: "v25.3.3",
			Source:  "https://helm.ngc.nvidia.com/nvidia",
		},
	}

	_, err = deployer.Generate(ctx, recipeResult, tmpDir)
	if err != nil {
		t.Fatalf("Generate() failed: %v", err)
	}

	// Read and verify the Application manifest from component directory
	appPath := filepath.Join(tmpDir, "gpu-operator", "argocd", "application.yaml")
	content, err := os.ReadFile(appPath)
	if err != nil {
		t.Fatalf("Failed to read Application manifest: %v", err)
	}

	// Verify Application YAML contains expected fields
	expectedStrings := []string{
		"apiVersion: argoproj.io/v1alpha1",
		"kind: Application",
		"gpu-operator",
		"valueFiles:",
		"gpu-operator/values.yaml",
	}

	for _, expected := range expectedStrings {
		if !strings.Contains(string(content), expected) {
			t.Errorf("Application manifest does not contain expected string: %s", expected)
		}
	}
}

func TestInit_Registers(t *testing.T) {
	// Verify that the init() function registered the argocd deployer
	r := registry.NewFromGlobal()

	deployer, found := r.Get(types.DeployerTypeArgoCD)
	if !found {
		t.Error("ArgoCD deployer not found in global registry after init()")
	}

	if deployer == nil {
		t.Error("ArgoCD deployer is nil in global registry")
	}
}

func TestDeployer_Generate_DeploymentOrder(t *testing.T) {
	deployer := &Deployer{}
	ctx := context.Background()

	tmpDir, err := os.MkdirTemp("", "argocd-order-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create recipe with components in reverse order but deployment order specified
	recipeResult := &recipe.RecipeResult{}
	recipeResult.Metadata.Version = testVersion
	recipeResult.ComponentRefs = []recipe.ComponentRef{
		{Name: "skyhook", Version: "v1.0.0", Source: "https://example.com"},
		{Name: "gpu-operator", Version: "v25.3.3", Source: "https://example.com"},
		{Name: "cert-manager", Version: "v1.14.0", Source: "https://example.com"},
	}
	// Deployment order: cert-manager first, then gpu-operator, then skyhook
	recipeResult.DeploymentOrder = []string{"cert-manager", "gpu-operator", "skyhook"}

	artifacts, err := deployer.Generate(ctx, recipeResult, tmpDir)
	if err != nil {
		t.Fatalf("Generate() failed: %v", err)
	}

	if !artifacts.Success {
		t.Fatalf("Generate() failed: %s", artifacts.Error)
	}

	// Read and verify sync-wave annotations
	testCases := []struct {
		component    string
		expectedWave string
	}{
		{"cert-manager", "0"},
		{"gpu-operator", "1"},
		{"skyhook", "2"},
	}

	for _, tc := range testCases {
		appPath := filepath.Join(tmpDir, tc.component, "argocd", "application.yaml")
		content, err := os.ReadFile(appPath)
		if err != nil {
			t.Fatalf("Failed to read %s: %v", appPath, err)
		}

		// Verify sync-wave annotation is present with correct value
		expectedAnnotation := "argocd.argoproj.io/sync-wave: \"" + tc.expectedWave + "\""
		if !strings.Contains(string(content), expectedAnnotation) {
			t.Errorf("Application %s does not have correct sync-wave annotation.\nExpected: %s\nContent:\n%s",
				tc.component, expectedAnnotation, string(content))
		}
	}
}

func TestOrderComponentsByDeployment(t *testing.T) {
	tests := []struct {
		name       string
		components []recipe.ComponentRef
		order      []string
		wantOrder  []string
	}{
		{
			name: "orders by deployment order",
			components: []recipe.ComponentRef{
				{Name: "c"},
				{Name: "b"},
				{Name: "a"},
			},
			order:     []string{"a", "b", "c"},
			wantOrder: []string{"a", "b", "c"},
		},
		{
			name: "empty order returns original",
			components: []recipe.ComponentRef{
				{Name: "c"},
				{Name: "b"},
				{Name: "a"},
			},
			order:     []string{},
			wantOrder: []string{"c", "b", "a"},
		},
		{
			name: "components not in order go last",
			components: []recipe.ComponentRef{
				{Name: "unknown"},
				{Name: "b"},
				{Name: "a"},
			},
			order:     []string{"a", "b"},
			wantOrder: []string{"a", "b", "unknown"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := orderComponentsByDeployment(tt.components, tt.order)

			if len(result) != len(tt.wantOrder) {
				t.Fatalf("got %d components, want %d", len(result), len(tt.wantOrder))
			}

			for i, comp := range result {
				if comp.Name != tt.wantOrder[i] {
					t.Errorf("position %d: got %s, want %s", i, comp.Name, tt.wantOrder[i])
				}
			}
		})
	}
}

func TestDeployer_Generate_WithManifestsDirectory(t *testing.T) {
	deployer := &Deployer{}
	ctx := context.Background()

	tmpDir, err := os.MkdirTemp("", "argocd-manifests-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a manifests directory for gpu-operator (simulating bundler output)
	gpuManifestsDir := filepath.Join(tmpDir, "gpu-operator", "manifests")
	if err = os.MkdirAll(gpuManifestsDir, 0755); err != nil {
		t.Fatalf("Failed to create manifests dir: %v", err)
	}
	// Create a sample manifest file
	if err = os.WriteFile(filepath.Join(gpuManifestsDir, "clusterpolicy.yaml"), []byte("test"), 0600); err != nil {
		t.Fatalf("Failed to create manifest file: %v", err)
	}

	recipeResult := &recipe.RecipeResult{}
	recipeResult.Metadata.Version = testVersion
	recipeResult.ComponentRefs = []recipe.ComponentRef{
		{
			Name:    "gpu-operator",
			Version: "v25.3.3",
			Source:  "https://helm.ngc.nvidia.com/nvidia",
		},
		{
			Name:    "cert-manager",
			Version: "v1.14.0",
			Source:  "https://charts.jetstack.io",
		},
	}

	artifacts, err := deployer.Generate(ctx, recipeResult, tmpDir)
	if err != nil {
		t.Fatalf("Generate() failed: %v", err)
	}

	if !artifacts.Success {
		t.Fatalf("Generate() failed: %s", artifacts.Error)
	}

	// Verify gpu-operator Application includes manifests source
	gpuAppPath := filepath.Join(tmpDir, "gpu-operator", "argocd", "application.yaml")
	gpuContent, err := os.ReadFile(gpuAppPath)
	if err != nil {
		t.Fatalf("Failed to read gpu-operator Application: %v", err)
	}

	if !strings.Contains(string(gpuContent), "gpu-operator/manifests") {
		t.Error("gpu-operator Application should include manifests directory source")
	}

	// Verify cert-manager Application does NOT include manifests source (no manifests dir)
	certAppPath := filepath.Join(tmpDir, "cert-manager", "argocd", "application.yaml")
	certContent, err := os.ReadFile(certAppPath)
	if err != nil {
		t.Fatalf("Failed to read cert-manager Application: %v", err)
	}

	if strings.Contains(string(certContent), "cert-manager/manifests") {
		t.Error("cert-manager Application should NOT include manifests directory source (none exists)")
	}
}

func TestAppOfAppsData_Fields(t *testing.T) {
	data := AppOfAppsData{
		Components: []AppOfAppsComponent{
			{Name: "gpu-operator", Namespace: "gpu-operator", Path: "gpu-operator/argocd"},
			{Name: "cert-manager", Namespace: "cert-manager", Path: "cert-manager/argocd"},
		},
	}

	if len(data.Components) != 2 {
		t.Errorf("AppOfAppsData.Components length = %d, want 2", len(data.Components))
	}

	if data.Components[0].Name != "gpu-operator" {
		t.Errorf("AppOfAppsData.Components[0].Name = %s, want gpu-operator", data.Components[0].Name)
	}

	if data.Components[0].Path != "gpu-operator/argocd" {
		t.Errorf("AppOfAppsData.Components[0].Path = %s, want gpu-operator/argocd", data.Components[0].Path)
	}
}
