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

package checksum

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

// ChecksumFileName is the standard name for checksum files.
const ChecksumFileName = "checksums.txt"

// GenerateChecksums creates a checksums.txt file containing SHA256 checksums
// for all provided files. The checksums are written relative to the bundle directory.
//
// Parameters:
//   - ctx: Context for cancellation
//   - bundleDir: The base directory for relative path calculation
//   - files: List of absolute file paths to include in checksums
//
// Returns an error if the context is canceled, any file cannot be read,
// or the checksums file cannot be written.
func GenerateChecksums(ctx context.Context, bundleDir string, files []string) error {
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("context cancelled: %w", err)
	}

	checksums := make([]string, 0, len(files))

	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			return fmt.Errorf("failed to read %s for checksum: %w", file, err)
		}

		hash := sha256.Sum256(data)
		relPath, err := filepath.Rel(bundleDir, file)
		if err != nil {
			// If relative path fails, use absolute path
			relPath = file
		}

		checksums = append(checksums, fmt.Sprintf("%s  %s", hex.EncodeToString(hash[:]), relPath))
	}

	checksumPath := filepath.Join(bundleDir, ChecksumFileName)
	content := strings.Join(checksums, "\n") + "\n"

	if err := os.WriteFile(checksumPath, []byte(content), 0600); err != nil {
		return fmt.Errorf("failed to write checksums: %w", err)
	}

	slog.Debug("checksums generated",
		"file_count", len(checksums),
		"path", checksumPath,
	)

	return nil
}

// GetChecksumFilePath returns the full path to the checksums.txt file
// in the given bundle directory.
func GetChecksumFilePath(bundleDir string) string {
	return filepath.Join(bundleDir, ChecksumFileName)
}
