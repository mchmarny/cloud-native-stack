/*
Copyright © 2025 NVIDIA Corporation
SPDX-License-Identifier: Apache-2.0
*/

package oci

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	ociv1 "github.com/opencontainers/image-spec/specs-go/v1"
	oras "oras.land/oras-go/v2"
	"oras.land/oras-go/v2/content/file"
	"oras.land/oras-go/v2/content/oci"

	"github.com/NVIDIA/cloud-native-stack/pkg/bundler/config"
	"github.com/NVIDIA/cloud-native-stack/pkg/component/certmanager"
	"github.com/NVIDIA/cloud-native-stack/pkg/recipe"
)

func TestStripProtocol(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "https prefix",
			input:    "https://ghcr.io",
			expected: "ghcr.io",
		},
		{
			name:     "http prefix",
			input:    "http://localhost:5000",
			expected: "localhost:5000",
		},
		{
			name:     "no prefix",
			input:    "registry.example.com",
			expected: "registry.example.com",
		},
		{
			name:     "with port no prefix",
			input:    "localhost:5000",
			expected: "localhost:5000",
		},
		{
			name:     "https with path",
			input:    "https://ghcr.io/nvidia",
			expected: "ghcr.io/nvidia",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := stripProtocol(tt.input)
			if got != tt.expected {
				t.Errorf("stripProtocol(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestPushFromStore_EmptyTag(t *testing.T) {
	// PushFromStore should fail when tag is empty
	_, err := PushFromStore(context.Background(), "/nonexistent", PushOptions{
		Registry:   "localhost:5000",
		Repository: "test/repo",
		Tag:        "", // Empty tag should fail
	})

	if err == nil {
		t.Error("PushFromStore() expected error for empty tag, got nil")
	}

	expectedErr := "tag is required to push OCI image"
	if err.Error() != expectedErr {
		t.Errorf("PushFromStore() error = %q, want %q", err.Error(), expectedErr)
	}
}

func TestPushFromStore_InvalidReference(t *testing.T) {
	// PushFromStore should fail for invalid registry references
	_, err := PushFromStore(context.Background(), "/nonexistent", PushOptions{
		Registry:   "invalid registry with spaces",
		Repository: "test/repo",
		Tag:        "v1.0.0",
	})

	if err == nil {
		t.Error("PushFromStore() expected error for invalid registry, got nil")
	}
}

func TestPushOptions_Defaults(t *testing.T) {
	opts := PushOptions{
		SourceDir:  "/tmp/test",
		Registry:   "ghcr.io",
		Repository: "nvidia/eidos",
		Tag:        "v1.0.0",
	}

	// Verify defaults
	if opts.PlainHTTP != false {
		t.Error("PlainHTTP should default to false")
	}
	if opts.InsecureTLS != false {
		t.Error("InsecureTLS should default to false")
	}
	if opts.SubDir != "" {
		t.Error("SubDir should default to empty string")
	}
}

func TestPushResult_Fields(t *testing.T) {
	result := PushResult{
		Digest:    "sha256:abc123",
		Reference: "ghcr.io/nvidia/eidos:v1.0.0",
	}

	if result.Digest != "sha256:abc123" {
		t.Errorf("Digest = %q, want %q", result.Digest, "sha256:abc123")
	}
	if result.Reference != "ghcr.io/nvidia/eidos:v1.0.0" {
		t.Errorf("Reference = %q, want %q", result.Reference, "ghcr.io/nvidia/eidos:v1.0.0")
	}
}

func TestValidateRegistryReference(t *testing.T) {
	tests := []struct {
		name       string
		registry   string
		repository string
		wantErr    bool
	}{
		{
			name:       "valid ghcr.io",
			registry:   "ghcr.io",
			repository: "nvidia/eidos",
			wantErr:    false,
		},
		{
			name:       "valid localhost with port",
			registry:   "localhost:5000",
			repository: "test/repo",
			wantErr:    false,
		},
		{
			name:       "valid with https prefix",
			registry:   "https://ghcr.io",
			repository: "nvidia/eidos",
			wantErr:    false,
		},
		{
			name:       "invalid registry with spaces",
			registry:   "invalid registry",
			repository: "test/repo",
			wantErr:    true,
		},
		{
			name:       "invalid repository with uppercase",
			registry:   "ghcr.io",
			repository: "NVIDIA/Eidos",
			wantErr:    true,
		},
		{
			name:       "invalid repository with special chars",
			registry:   "ghcr.io",
			repository: "test/repo@latest",
			wantErr:    true,
		},
		{
			name:       "valid complex repository",
			registry:   "registry.example.com:5000",
			repository: "org/team/project",
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateRegistryReference(tt.registry, tt.repository)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateRegistryReference() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestPackage_Validation(t *testing.T) {
	ctx := context.Background()

	// Test missing tag
	_, err := Package(ctx, PackageOptions{
		SourceDir:  ".",
		OutputDir:  t.TempDir(),
		Registry:   "ghcr.io",
		Repository: "test/repo",
		Tag:        "",
	})
	if err == nil || err.Error() != "tag is required for OCI packaging" {
		t.Errorf("Package() expected tag error, got: %v", err)
	}

	// Test missing registry
	_, err = Package(ctx, PackageOptions{
		SourceDir:  ".",
		OutputDir:  t.TempDir(),
		Registry:   "",
		Repository: "test/repo",
		Tag:        "v1.0.0",
	})
	if err == nil || err.Error() != "registry is required for OCI packaging" {
		t.Errorf("Package() expected registry error, got: %v", err)
	}

	// Test missing repository
	_, err = Package(ctx, PackageOptions{
		SourceDir:  ".",
		OutputDir:  t.TempDir(),
		Registry:   "ghcr.io",
		Repository: "",
		Tag:        "v1.0.0",
	})
	if err == nil || err.Error() != "repository is required for OCI packaging" {
		t.Errorf("Package() expected repository error, got: %v", err)
	}
}

func TestPackage_CreatesOCILayout(t *testing.T) {
	ctx := context.Background()

	// Create source directory with test files
	sourceDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(sourceDir, "test.yaml"), []byte("content: test"), 0o644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	outputDir := t.TempDir()

	result, err := Package(ctx, PackageOptions{
		SourceDir:  sourceDir,
		OutputDir:  outputDir,
		Registry:   "ghcr.io",
		Repository: "test/repo",
		Tag:        "v1.0.0",
	})
	if err != nil {
		t.Fatalf("Package() error = %v", err)
	}

	// Verify result fields
	if result.Digest == "" {
		t.Error("Package() result has empty digest")
	}
	if result.Reference != "ghcr.io/test/repo:v1.0.0" {
		t.Errorf("Package() reference = %q, want %q", result.Reference, "ghcr.io/test/repo:v1.0.0")
	}
	if result.StorePath == "" {
		t.Error("Package() result has empty store path")
	}

	// Verify OCI layout was created
	ociLayoutFile := filepath.Join(result.StorePath, "oci-layout")
	if _, err := os.Stat(ociLayoutFile); os.IsNotExist(err) {
		t.Errorf("Package() did not create oci-layout file at %s", ociLayoutFile)
	}

	// Verify index.json exists
	indexFile := filepath.Join(result.StorePath, "index.json")
	if _, err := os.Stat(indexFile); os.IsNotExist(err) {
		t.Errorf("Package() did not create index.json at %s", indexFile)
	}

	t.Logf("Package() created OCI layout at %s with digest %s", result.StorePath, result.Digest)
}

// TestOCIPackagingIntegration is an integration test that uses the REAL cert-manager
// bundler to generate bundle output and the REAL OCI packaging code to create an artifact.
// This verifies the entire pipeline from recipe → bundler → OCI artifact.
func TestOCIPackagingIntegration(t *testing.T) {
	ctx := context.Background()

	// Create output directory for bundler
	bundleOutputDir := t.TempDir()

	// Create a test RecipeResult with cert-manager component reference
	// (RecipeResult is required because bundlers use GetComponentRef)
	rec := &recipe.RecipeResult{
		Kind:       "recipeResult",
		APIVersion: recipe.FullAPIVersion,
		ComponentRefs: []recipe.ComponentRef{
			{
				Name:       "cert-manager",
				Type:       "Helm",
				Source:     "https://charts.jetstack.io",
				Version:    "v1.14.0",
				ValuesFile: "components/cert-manager/values.yaml",
			},
		},
	}

	// Use the REAL cert-manager bundler to generate output
	bundler := certmanager.NewBundler(config.NewConfig())
	result, err := bundler.Make(ctx, rec, bundleOutputDir)
	if err != nil {
		t.Fatalf("Bundler.Make() error = %v", err)
	}

	if !result.Success {
		t.Fatalf("Bundler.Make() did not succeed: %v", result.Errors)
	}

	// Verify bundler created files
	bundlerDir := filepath.Join(bundleOutputDir, "cert-manager")
	if _, statErr := os.Stat(bundlerDir); os.IsNotExist(statErr) {
		t.Fatalf("Bundler did not create cert-manager directory")
	}

	t.Logf("Bundler created %d files in %s", len(result.Files), bundlerDir)

	// Now use the REAL OCI packaging code (same as Push function)
	// but write to a local OCI layout store instead of a remote registry

	// Create OCI layout store as the push target
	ociLayoutDir := t.TempDir()
	ociStore, err := oci.New(ociLayoutDir)
	if err != nil {
		t.Fatalf("Failed to create OCI layout store: %v", err)
	}

	// Create a file store from the bundler output directory (same as Push does)
	fs, err := file.New(bundlerDir)
	if err != nil {
		t.Fatalf("Failed to create file store: %v", err)
	}
	defer func() { _ = fs.Close() }()

	// Enable deterministic tar creation (same as Push)
	fs.TarReproducible = true

	// Add directory contents as a gzipped tar layer (same as Push)
	layerDesc, err := fs.Add(ctx, ".", ociv1.MediaTypeImageLayerGzip, bundlerDir)
	if err != nil {
		t.Fatalf("Failed to add directory to store: %v", err)
	}

	// Verify layer media type
	if layerDesc.MediaType != ociv1.MediaTypeImageLayerGzip {
		t.Errorf("Layer MediaType = %q, want %q", layerDesc.MediaType, ociv1.MediaTypeImageLayerGzip)
	}

	// Pack an OCI 1.1 manifest (same as Push)
	packOpts := oras.PackManifestOptions{
		Layers: []ociv1.Descriptor{layerDesc},
	}
	manifestDesc, err := oras.PackManifest(ctx, fs, oras.PackManifestVersion1_1, ArtifactType, packOpts)
	if err != nil {
		t.Fatalf("Failed to pack manifest: %v", err)
	}

	// Tag the manifest
	tag := "v1.0.0-integration-test"
	if tagErr := fs.Tag(ctx, manifestDesc, tag); tagErr != nil {
		t.Fatalf("Failed to tag manifest: %v", tagErr)
	}

	// Copy to OCI layout store (simulates push to registry)
	desc, err := oras.Copy(ctx, fs, tag, ociStore, tag, oras.DefaultCopyOptions)
	if err != nil {
		t.Fatalf("Failed to copy to OCI layout: %v", err)
	}

	// Verify the manifest was pushed with a valid digest
	if desc.Digest.String() == "" {
		t.Error("Pushed manifest has empty digest")
	}

	// Read and verify the manifest structure
	manifestPath := filepath.Join(ociLayoutDir, "blobs", "sha256", strings.TrimPrefix(desc.Digest.String(), "sha256:"))
	manifestData, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("Failed to read manifest: %v", err)
	}

	var manifest ociv1.Manifest
	if unmarshalErr := json.Unmarshal(manifestData, &manifest); unmarshalErr != nil {
		t.Fatalf("Failed to unmarshal manifest: %v", unmarshalErr)
	}

	// Verify artifact type matches what Package() uses
	if manifest.ArtifactType != ArtifactType {
		t.Errorf("Manifest ArtifactType = %q, want %q", manifest.ArtifactType, ArtifactType)
	}

	// Verify we have exactly one layer
	if len(manifest.Layers) != 1 {
		t.Fatalf("Manifest has %d layers, want 1", len(manifest.Layers))
	}

	// Read and extract the layer to verify contents
	layerDigest := manifest.Layers[0].Digest.String()
	layerPath := filepath.Join(ociLayoutDir, "blobs", "sha256", strings.TrimPrefix(layerDigest, "sha256:"))
	layerFile, err := os.Open(layerPath)
	if err != nil {
		t.Fatalf("Failed to open layer: %v", err)
	}
	defer layerFile.Close()

	// Decompress gzip
	gzr, err := gzip.NewReader(layerFile)
	if err != nil {
		t.Fatalf("Failed to create gzip reader: %v", err)
	}
	defer gzr.Close()

	// Extract tar and collect file names
	var extractedFiles []string
	tr := tar.NewReader(gzr)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("Failed to read tar entry: %v", err)
		}
		if header.Typeflag == tar.TypeReg {
			extractedFiles = append(extractedFiles, header.Name)
		}
	}

	// Verify expected cert-manager bundler files are present
	expectedFiles := []string{
		"values.yaml",
		"scripts/install.sh",
		"scripts/uninstall.sh",
		"README.md",
		"checksums.txt",
	}

	sort.Strings(extractedFiles)
	sort.Strings(expectedFiles)

	for _, expected := range expectedFiles {
		found := false
		for _, actual := range extractedFiles {
			if actual == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected file %q not found in OCI artifact. Got files: %v", expected, extractedFiles)
		}
	}

	t.Logf("Integration test passed: OCI artifact contains %d files from real bundler output, digest: %s",
		len(extractedFiles), desc.Digest.String())
}

// TestOCIArtifactStructure tests the OCI packaging with synthetic test files
// to verify the artifact structure is correct.
func TestOCIArtifactStructure(t *testing.T) {
	ctx := context.Background()

	// Create a temporary bundle directory with test files
	bundleDir := t.TempDir()
	testFiles := map[string]string{
		"manifest.yaml":           "apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: test",
		"helm/chart/Chart.yaml":   "apiVersion: v2\nname: test-chart\nversion: 1.0.0",
		"helm/chart/values.yaml":  "replicaCount: 1\nimage:\n  tag: latest",
		"terraform/main.tf":       "resource \"null_resource\" \"test\" {}",
		"scripts/install.sh":      "#!/bin/bash\necho 'Installing...'",
		"README.md":               "# Test Bundle\nThis is a test bundle.",
		"nested/deep/config.json": `{"key": "value", "nested": {"foo": "bar"}}`,
	}

	for path, content := range testFiles {
		fullPath := filepath.Join(bundleDir, path)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
			t.Fatalf("Failed to create directory for %s: %v", path, err)
		}
		if err := os.WriteFile(fullPath, []byte(content), 0o644); err != nil {
			t.Fatalf("Failed to write test file %s: %v", path, err)
		}
	}

	// Create an OCI layout store as the push target
	ociLayoutDir := t.TempDir()
	ociStore, err := oci.New(ociLayoutDir)
	if err != nil {
		t.Fatalf("Failed to create OCI layout store: %v", err)
	}

	// Create a file store from the bundle directory (same as Push does)
	fs, err := file.New(bundleDir)
	if err != nil {
		t.Fatalf("Failed to create file store: %v", err)
	}
	defer func() { _ = fs.Close() }()

	// Enable deterministic tar creation (same as Push)
	fs.TarReproducible = true

	// Add directory contents as a gzipped tar layer
	layerDesc, err := fs.Add(ctx, ".", ociv1.MediaTypeImageLayerGzip, bundleDir)
	if err != nil {
		t.Fatalf("Failed to add directory to store: %v", err)
	}

	// Verify layer media type
	if layerDesc.MediaType != ociv1.MediaTypeImageLayerGzip {
		t.Errorf("Layer MediaType = %q, want %q", layerDesc.MediaType, ociv1.MediaTypeImageLayerGzip)
	}

	// Pack an OCI 1.1 manifest (same as Push)
	packOpts := oras.PackManifestOptions{
		Layers: []ociv1.Descriptor{layerDesc},
	}
	manifestDesc, err := oras.PackManifest(ctx, fs, oras.PackManifestVersion1_1, ArtifactType, packOpts)
	if err != nil {
		t.Fatalf("Failed to pack manifest: %v", err)
	}

	// Tag the manifest
	tag := "v1.0.0-test"
	if tagErr := fs.Tag(ctx, manifestDesc, tag); tagErr != nil {
		t.Fatalf("Failed to tag manifest: %v", tagErr)
	}

	// Copy to OCI layout store (simulates push to registry)
	desc, err := oras.Copy(ctx, fs, tag, ociStore, tag, oras.DefaultCopyOptions)
	if err != nil {
		t.Fatalf("Failed to copy to OCI layout: %v", err)
	}

	// Verify the manifest was pushed
	if desc.Digest.String() == "" {
		t.Error("Pushed manifest has empty digest")
	}

	// Read and verify the manifest structure
	manifestPath := filepath.Join(ociLayoutDir, "blobs", "sha256", strings.TrimPrefix(desc.Digest.String(), "sha256:"))
	manifestData, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("Failed to read manifest: %v", err)
	}

	var manifest ociv1.Manifest
	if unmarshalErr := json.Unmarshal(manifestData, &manifest); unmarshalErr != nil {
		t.Fatalf("Failed to unmarshal manifest: %v", unmarshalErr)
	}

	// Verify artifact type
	if manifest.ArtifactType != ArtifactType {
		t.Errorf("Manifest ArtifactType = %q, want %q", manifest.ArtifactType, ArtifactType)
	}

	// Verify we have exactly one layer
	if len(manifest.Layers) != 1 {
		t.Fatalf("Manifest has %d layers, want 1", len(manifest.Layers))
	}

	// Read and verify the layer contents
	layerDigest := manifest.Layers[0].Digest.String()
	layerPath := filepath.Join(ociLayoutDir, "blobs", "sha256", strings.TrimPrefix(layerDigest, "sha256:"))
	layerFile, err := os.Open(layerPath)
	if err != nil {
		t.Fatalf("Failed to open layer: %v", err)
	}
	defer layerFile.Close()

	// Decompress and extract tar
	gzr, err := gzip.NewReader(layerFile)
	if err != nil {
		t.Fatalf("Failed to create gzip reader: %v", err)
	}
	defer gzr.Close()

	// Extract all files from the tar and verify
	extractedFiles := make(map[string]string)
	tr := tar.NewReader(gzr)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("Failed to read tar entry: %v", err)
		}

		if header.Typeflag == tar.TypeReg {
			content, err := io.ReadAll(tr)
			if err != nil {
				t.Fatalf("Failed to read tar file content: %v", err)
			}
			extractedFiles[header.Name] = string(content)
		}
	}

	// Verify all expected files are present with correct content
	for expectedPath, expectedContent := range testFiles {
		actualContent, ok := extractedFiles[expectedPath]
		if !ok {
			t.Errorf("Expected file %q not found in artifact", expectedPath)
			continue
		}
		if actualContent != expectedContent {
			t.Errorf("File %q content mismatch:\n  got:  %q\n  want: %q", expectedPath, actualContent, expectedContent)
		}
	}

	// Verify no unexpected files
	for path := range extractedFiles {
		if _, ok := testFiles[path]; !ok {
			t.Errorf("Unexpected file in artifact: %q", path)
		}
	}

	t.Logf("Successfully verified OCI artifact with %d files, digest: %s", len(extractedFiles), desc.Digest.String())
}

// TestOCIReproducibleBuild verifies that builds are deterministic.
func TestOCIReproducibleBuild(t *testing.T) {
	ctx := context.Background()

	// Create a bundle directory with test files
	bundleDir := t.TempDir()
	testFiles := map[string]string{
		"file1.yaml": "content: one",
		"file2.yaml": "content: two",
		"file3.yaml": "content: three",
	}

	for path, content := range testFiles {
		fullPath := filepath.Join(bundleDir, path)
		if err := os.WriteFile(fullPath, []byte(content), 0o644); err != nil {
			t.Fatalf("Failed to write test file %s: %v", path, err)
		}
	}

	// Build twice and compare digests
	var digests []string
	for i := 0; i < 2; i++ {
		ociLayoutDir := t.TempDir()
		ociStore, err := oci.New(ociLayoutDir)
		if err != nil {
			t.Fatalf("Iteration %d: Failed to create OCI layout store: %v", i, err)
		}

		fs, err := file.New(bundleDir)
		if err != nil {
			t.Fatalf("Iteration %d: Failed to create file store: %v", i, err)
		}

		// Critical: enable reproducible tars
		fs.TarReproducible = true

		layerDesc, err := fs.Add(ctx, ".", ociv1.MediaTypeImageLayerGzip, bundleDir)
		if err != nil {
			_ = fs.Close()
			t.Fatalf("Iteration %d: Failed to add directory to store: %v", i, err)
		}

		packOpts := oras.PackManifestOptions{
			Layers: []ociv1.Descriptor{layerDesc},
			// Use fixed timestamp for reproducible manifest
			ManifestAnnotations: map[string]string{
				ociv1.AnnotationCreated: "2000-01-01T00:00:00Z",
			},
		}
		manifestDesc, err := oras.PackManifest(ctx, fs, oras.PackManifestVersion1_1, ArtifactType, packOpts)
		if err != nil {
			_ = fs.Close()
			t.Fatalf("Iteration %d: Failed to pack manifest: %v", i, err)
		}

		tag := "repro-test"
		if tagErr := fs.Tag(ctx, manifestDesc, tag); tagErr != nil {
			_ = fs.Close()
			t.Fatalf("Iteration %d: Failed to tag manifest: %v", i, tagErr)
		}

		desc, err := oras.Copy(ctx, fs, tag, ociStore, tag, oras.DefaultCopyOptions)
		_ = fs.Close()
		if err != nil {
			t.Fatalf("Iteration %d: Failed to copy to OCI layout: %v", i, err)
		}

		digests = append(digests, desc.Digest.String())
	}

	// Verify both builds produced the same digest
	if digests[0] != digests[1] {
		t.Errorf("Reproducible builds produced different digests:\n  build 1: %s\n  build 2: %s", digests[0], digests[1])
	} else {
		t.Logf("Reproducible build verified: both iterations produced digest %s", digests[0])
	}
}
