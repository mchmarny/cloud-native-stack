package internal

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/NVIDIA/cloud-native-stack/pkg/bundler/result"
	"github.com/NVIDIA/cloud-native-stack/pkg/bundler/types"
)

func TestTemplateRenderer_Render(t *testing.T) {
	templates := map[string]string{
		"test": "Hello {{.Name}}!",
	}

	getter := func(name string) (string, bool) {
		tmpl, ok := templates[name]
		return tmpl, ok
	}

	renderer := NewTemplateRenderer(getter)

	tests := []struct {
		name     string
		tmplName string
		data     map[string]interface{}
		want     string
		wantErr  bool
	}{
		{
			name:     "renders template",
			tmplName: "test",
			data:     map[string]interface{}{"Name": "World"},
			want:     "Hello World!",
			wantErr:  false,
		},
		{
			name:     "template not found",
			tmplName: "missing",
			data:     map[string]interface{}{},
			want:     "",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := renderer.Render(tt.tmplName, tt.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("Render() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Render() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDirectoryManager_CreateDirectories(t *testing.T) {
	tmpDir := t.TempDir()
	manager := NewDirectoryManager()

	dirs := []string{
		filepath.Join(tmpDir, "dir1"),
		filepath.Join(tmpDir, "dir2", "subdir"),
	}

	err := manager.CreateDirectories(dirs, 0755)
	if err != nil {
		t.Fatalf("CreateDirectories() error = %v", err)
	}

	// Verify directories were created
	for _, dir := range dirs {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			t.Errorf("Directory %s was not created", dir)
		}
	}
}

func TestDirectoryManager_CreateBundleStructure(t *testing.T) {
	tmpDir := t.TempDir()
	manager := NewDirectoryManager()

	bundleDir, subdirs, err := manager.CreateBundleStructure(tmpDir, "test-bundle")
	if err != nil {
		t.Fatalf("CreateBundleStructure() error = %v", err)
	}

	// Verify bundle directory
	expectedBundleDir := filepath.Join(tmpDir, "test-bundle")
	if bundleDir != expectedBundleDir {
		t.Errorf("bundleDir = %v, want %v", bundleDir, expectedBundleDir)
	}

	// Verify subdirectories
	expectedSubdirs := map[string]string{
		"root":      expectedBundleDir,
		"scripts":   filepath.Join(expectedBundleDir, "scripts"),
		"manifests": filepath.Join(expectedBundleDir, "manifests"),
	}

	for key, expectedPath := range expectedSubdirs {
		if subdirs[key] != expectedPath {
			t.Errorf("subdirs[%s] = %v, want %v", key, subdirs[key], expectedPath)
		}

		// Verify directory exists
		if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
			t.Errorf("Directory %s was not created", expectedPath)
		}
	}
}

func TestContextChecker_Check(t *testing.T) {
	checker := NewContextChecker()

	t.Run("active context", func(t *testing.T) {
		ctx := context.Background()
		err := checker.Check(ctx)
		if err != nil {
			t.Errorf("Check() with active context should not error, got %v", err)
		}
	})

	t.Run("cancelled context", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		err := checker.Check(ctx)
		if err == nil {
			t.Error("Check() with cancelled context should error")
		}
	})

	t.Run("expired context", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
		defer cancel()
		time.Sleep(10 * time.Millisecond)

		err := checker.Check(ctx)
		if err == nil {
			t.Error("Check() with expired context should error")
		}
	})
}

func TestComputeChecksum(t *testing.T) {
	tests := []struct {
		name    string
		content []byte
		want    string
	}{
		{
			name:    "empty content",
			content: []byte{},
			want:    "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
		},
		{
			name:    "hello world",
			content: []byte("hello world"),
			want:    "b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ComputeChecksum(tt.content)
			if got != tt.want {
				t.Errorf("ComputeChecksum() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFileWriter(t *testing.T) {
	tmpDir := t.TempDir()
	res := result.New(types.BundleTypeGpuOperator)
	writer := NewFileWriter(res)

	t.Run("WriteFile", func(t *testing.T) {
		filePath := filepath.Join(tmpDir, "test.txt")
		data := []byte("test content")

		err := writer.WriteFile(filePath, data, 0644)
		if err != nil {
			t.Fatalf("WriteFile() error = %v", err)
		}

		// Verify file exists and has correct content
		content, err := os.ReadFile(filePath)
		if err != nil {
			t.Fatalf("Failed to read file: %v", err)
		}
		if string(content) != "test content" {
			t.Errorf("File content = %s, want 'test content'", string(content))
		}

		// Verify file permissions
		info, err := os.Stat(filePath)
		if err != nil {
			t.Fatalf("Failed to stat file: %v", err)
		}
		if info.Mode().Perm() != 0644 {
			t.Errorf("File permissions = %o, want 0644", info.Mode().Perm())
		}

		// Verify file was added to result
		if len(res.Files) != 1 {
			t.Errorf("Number of files in result = %d, want 1", len(res.Files))
		}
	})

	t.Run("WriteFileString", func(t *testing.T) {
		filePath := filepath.Join(tmpDir, "test2.txt")
		content := "string content"

		err := writer.WriteFileString(filePath, content, 0644)
		if err != nil {
			t.Fatalf("WriteFileString() error = %v", err)
		}

		// Verify file exists and has correct content
		data, err := os.ReadFile(filePath)
		if err != nil {
			t.Fatalf("Failed to read file: %v", err)
		}
		if string(data) != content {
			t.Errorf("File content = %s, want %s", string(data), content)
		}
	})

	t.Run("MakeExecutable", func(t *testing.T) {
		filePath := filepath.Join(tmpDir, "script.sh")

		// Create a test file first
		err := os.WriteFile(filePath, []byte("#!/bin/bash\necho test"), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		// Make it executable
		err = writer.MakeExecutable(filePath)
		if err != nil {
			t.Fatalf("MakeExecutable() error = %v", err)
		}

		// Verify file is executable
		info, err := os.Stat(filePath)
		if err != nil {
			t.Fatalf("Failed to stat file: %v", err)
		}

		mode := info.Mode()
		if mode.Perm()&0111 == 0 {
			t.Errorf("File is not executable, mode = %o", mode.Perm())
		}
	})

	t.Run("MakeExecutable non-existent file", func(t *testing.T) {
		filePath := filepath.Join(tmpDir, "nonexistent.sh")

		err := writer.MakeExecutable(filePath)
		if err == nil {
			t.Error("MakeExecutable() should return error for non-existent file")
		}
	})
}

func TestChecksumGenerator(t *testing.T) {
	tmpDir := t.TempDir()

	t.Run("Generate with multiple files", func(t *testing.T) {
		// Create result with test files
		res := result.New(types.BundleTypeGpuOperator)

		// Create test files
		file1 := filepath.Join(tmpDir, "file1.txt")
		file2 := filepath.Join(tmpDir, "file2.txt")

		err := os.WriteFile(file1, []byte("content1"), 0644)
		if err != nil {
			t.Fatalf("Failed to create file1: %v", err)
		}
		err = os.WriteFile(file2, []byte("content2"), 0644)
		if err != nil {
			t.Fatalf("Failed to create file2: %v", err)
		}

		// Add files to result
		res.AddFile(file1, 8)
		res.AddFile(file2, 8)

		generator := NewChecksumGenerator(res)
		content, err := generator.Generate(tmpDir, "Test")

		if err != nil {
			t.Fatalf("Generate() error = %v", err)
		}

		// Verify content contains checksums for both files
		if !contains(content, "file1.txt") {
			t.Error("Generated checksums should contain file1.txt")
		}
		if !contains(content, "file2.txt") {
			t.Error("Generated checksums should contain file2.txt")
		}
		if !contains(content, "# Test Bundle Checksums") {
			t.Error("Generated checksums should contain title header")
		}
	})

	t.Run("Generate with empty result", func(t *testing.T) {
		res := result.New(types.BundleTypeGpuOperator)
		generator := NewChecksumGenerator(res)

		content, err := generator.Generate(tmpDir, "Empty")

		if err != nil {
			t.Fatalf("Generate() error = %v", err)
		}

		// Should only contain header
		if !contains(content, "# Empty Bundle Checksums") {
			t.Error("Generated checksums should contain header")
		}
	})

	t.Run("Generate with non-existent file in result", func(t *testing.T) {
		res := result.New(types.BundleTypeGpuOperator)
		// Add a file that doesn't exist
		res.AddFile(filepath.Join(tmpDir, "nonexistent.txt"), 0)

		generator := NewChecksumGenerator(res)
		_, err := generator.Generate(tmpDir, "Test")

		if err == nil {
			t.Error("Generate() should return error for non-existent file")
		}
	})
}

func TestGetConfigValue(t *testing.T) {
	tests := []struct {
		name         string
		config       map[string]string
		key          string
		defaultValue string
		want         string
	}{
		{
			name:         "key exists",
			config:       map[string]string{"key": "value"},
			key:          "key",
			defaultValue: "default",
			want:         "value",
		},
		{
			name:         "key missing",
			config:       map[string]string{},
			key:          "key",
			defaultValue: "default",
			want:         "default",
		},
		{
			name:         "empty value uses default",
			config:       map[string]string{"key": ""},
			key:          "key",
			defaultValue: "default",
			want:         "default",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetConfigValue(tt.config, tt.key, tt.defaultValue)
			if got != tt.want {
				t.Errorf("GetConfigValue() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestExtractCustomLabels(t *testing.T) {
	tests := []struct {
		name   string
		config map[string]string
		want   map[string]string
	}{
		{
			name: "extracts labels",
			config: map[string]string{
				"label_env":  "prod",
				"label_team": "platform",
				"other_key":  "value",
			},
			want: map[string]string{
				"env":  "prod",
				"team": "platform",
			},
		},
		{
			name:   "empty config",
			config: map[string]string{},
			want:   map[string]string{},
		},
		{
			name: "no labels",
			config: map[string]string{
				"key": "value",
			},
			want: map[string]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractCustomLabels(tt.config)
			if len(got) != len(tt.want) {
				t.Errorf("ExtractCustomLabels() len = %v, want %v", len(got), len(tt.want))
			}
			for k, v := range tt.want {
				if got[k] != v {
					t.Errorf("ExtractCustomLabels()[%v] = %v, want %v", k, got[k], v)
				}
			}
		})
	}
}

func TestExtractCustomAnnotations(t *testing.T) {
	tests := []struct {
		name   string
		config map[string]string
		want   map[string]string
	}{
		{
			name: "extracts annotations",
			config: map[string]string{
				"annotation_key1": "value1",
				"annotation_key2": "value2",
				"other_key":       "value",
			},
			want: map[string]string{
				"key1": "value1",
				"key2": "value2",
			},
		},
		{
			name:   "empty config",
			config: map[string]string{},
			want:   map[string]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractCustomAnnotations(tt.config)
			if len(got) != len(tt.want) {
				t.Errorf("ExtractCustomAnnotations() len = %v, want %v", len(got), len(tt.want))
			}
			for k, v := range tt.want {
				if got[k] != v {
					t.Errorf("ExtractCustomAnnotations()[%v] = %v, want %v", k, got[k], v)
				}
			}
		})
	}
}
