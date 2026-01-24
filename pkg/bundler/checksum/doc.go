/*
Copyright © 2025 NVIDIA Corporation
SPDX-License-Identifier: Apache-2.0
*/

// Package checksum provides SHA256 checksum generation for bundle verification.
//
// Used by component bundlers (GPU Operator, Network Operator, etc.) and deployers
// (Helm, ArgoCD) to generate checksums.txt files for integrity verification.
//
// Usage:
//
//	err := checksum.GenerateChecksums(ctx, "/path/to/bundle", fileList)
//	if err != nil {
//	    return err
//	}
//
// The checksums.txt file format is compatible with sha256sum:
//
//	sha256sum -c checksums.txt
package checksum
