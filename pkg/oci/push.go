/*
Copyright © 2025 NVIDIA Corporation
SPDX-License-Identifier: Apache-2.0
*/

// Package oci provides utilities for pushing artifacts to OCI registries.
package oci

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/distribution/reference"
	ociv1 "github.com/opencontainers/image-spec/specs-go/v1"
	oras "oras.land/oras-go/v2"
	"oras.land/oras-go/v2/content/file"
	"oras.land/oras-go/v2/registry/remote"
	"oras.land/oras-go/v2/registry/remote/auth"
	"oras.land/oras-go/v2/registry/remote/credentials"
)

// ArtifactType is the media type for eidos OCI artifacts.
const ArtifactType = "application/vnd.nvidia.eidos.artifact"

// PushOptions configures the OCI push operation.
type PushOptions struct {
	// SourceDir is the directory containing artifacts to push.
	SourceDir string
	// Registry is the OCI registry host (e.g., "ghcr.io", "localhost:5000").
	Registry string
	// Repository is the image repository path (e.g., "nvidia/eidos").
	Repository string
	// Tag is the image tag (e.g., "v1.0.0", "latest").
	Tag string
	// SubDir optionally limits the push to a subdirectory within SourceDir.
	SubDir string
	// PlainHTTP uses HTTP instead of HTTPS for the registry connection.
	PlainHTTP bool
	// InsecureTLS skips TLS certificate verification.
	InsecureTLS bool
	// ReproducibleTimestamp sets a fixed timestamp for reproducible builds.
	ReproducibleTimestamp string
}

// PushResult contains the result of a successful OCI push.
type PushResult struct {
	// Digest is the SHA256 digest of the pushed artifact.
	Digest string
	// Reference is the full image reference (registry/repository:tag).
	Reference string
}

// Push pushes an OCI artifact to a registry using ORAS.
func Push(ctx context.Context, opts PushOptions) (*PushResult, error) {
	if opts.Tag == "" {
		return nil, fmt.Errorf("tag is required to push OCI image")
	}

	// Determine the directory to push from
	pushFromDir, cleanup, err := preparePushDir(opts.SourceDir, opts.SubDir)
	if err != nil {
		return nil, err
	}
	if cleanup != nil {
		defer cleanup()
	}

	// Convert to absolute path to avoid ORAS working directory issues
	absPushDir, err := filepath.Abs(pushFromDir)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path for push dir: %w", err)
	}

	// Strip protocol from registry for docker reference compatibility
	registryHost := stripProtocol(opts.Registry)

	// Build and validate the image reference
	refString := fmt.Sprintf("%s/%s:%s", registryHost, opts.Repository, opts.Tag)
	if _, err := reference.ParseNormalizedNamed(refString); err != nil {
		return nil, fmt.Errorf("invalid image reference '%s': %w", refString, err)
	}

	// Create a file store rooted at the directory we want to push
	fs, err := file.New(absPushDir)
	if err != nil {
		return nil, fmt.Errorf("failed to create file store: %w", err)
	}
	defer func() { _ = fs.Close() }()

	// Make tars deterministic for reproducible builds
	fs.TarReproducible = true

	// Add all contents from the file store root
	layerDesc, err := fs.Add(ctx, ".", ociv1.MediaTypeImageLayerGzip, absPushDir)
	if err != nil {
		return nil, fmt.Errorf("failed to add source directory to store: %w", err)
	}

	// Pack an OCI 1.1 manifest with our artifact type
	packOpts := oras.PackManifestOptions{
		Layers: []ociv1.Descriptor{layerDesc},
	}

	// Attach reproducible created annotation if provided
	if opts.ReproducibleTimestamp != "" {
		packOpts.ManifestAnnotations = map[string]string{
			ociv1.AnnotationCreated: opts.ReproducibleTimestamp,
		}
	}

	manifestDesc, err := oras.PackManifest(ctx, fs, oras.PackManifestVersion1_1, ArtifactType, packOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to pack manifest: %w", err)
	}

	// Tag the local manifest so we can copy by tag
	if err := fs.Tag(ctx, manifestDesc, opts.Tag); err != nil {
		return nil, fmt.Errorf("failed to tag manifest in local store: %w", err)
	}

	// Prepare remote repository
	repo, err := remote.NewRepository(fmt.Sprintf("%s/%s", registryHost, opts.Repository))
	if err != nil {
		return nil, fmt.Errorf("failed to initialize remote repository: %w", err)
	}
	repo.PlainHTTP = opts.PlainHTTP

	// Configure auth client using Docker credentials if available
	repo.Client = createAuthClient(opts.PlainHTTP, opts.InsecureTLS)

	// Copy from the local file store to the remote repository
	desc, err := oras.Copy(ctx, fs, opts.Tag, repo, opts.Tag, oras.DefaultCopyOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to push artifact to registry: %w", err)
	}

	return &PushResult{
		Digest:    desc.Digest.String(),
		Reference: refString,
	}, nil
}

// preparePushDir prepares the directory for pushing.
// If subDir is specified, creates a temp directory with hard links.
// Returns the directory to push from and an optional cleanup function.
func preparePushDir(sourceDir, subDir string) (string, func(), error) {
	if subDir == "" {
		return sourceDir, nil, nil
	}

	// When pushing a subdirectory, preserve its path structure in the image
	// Create a temp dir and use hard links (fast, no extra disk space)
	tempDir, err := os.MkdirTemp("", "oras-push-*")
	if err != nil {
		return "", nil, fmt.Errorf("failed to create temp directory: %w", err)
	}

	srcPath := filepath.Join(sourceDir, subDir)
	dstPath := filepath.Join(tempDir, subDir)
	if err := hardLinkDir(srcPath, dstPath); err != nil {
		os.RemoveAll(tempDir)
		return "", nil, fmt.Errorf("failed to create hard links: %w", err)
	}

	cleanup := func() { os.RemoveAll(tempDir) }
	return tempDir, cleanup, nil
}

// stripProtocol removes http:// or https:// prefix from a registry URL.
func stripProtocol(registry string) string {
	registry = strings.TrimPrefix(registry, "https://")
	registry = strings.TrimPrefix(registry, "http://")
	return registry
}

// createAuthClient creates an HTTP client with optional TLS configuration
// and Docker credential support.
func createAuthClient(plainHTTP, insecureTLS bool) *auth.Client {
	credStore, _ := credentials.NewStoreFromDocker(credentials.StoreOptions{})

	transport := http.DefaultTransport.(*http.Transport).Clone()
	if !plainHTTP && insecureTLS {
		if transport.TLSClientConfig == nil {
			transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true} //nolint:gosec
		} else {
			transport.TLSClientConfig.InsecureSkipVerify = true //nolint:gosec
		}
	}

	return &auth.Client{
		Client:     &http.Client{Transport: transport},
		Cache:      auth.NewCache(),
		Credential: credentials.Credential(credStore),
	}
}

// hardLinkDir recursively creates hard links from src to dst.
// This is much faster than copying and uses no additional disk space.
func hardLinkDir(src, dst string) error {
	srcInfo, err := os.Stat(src)
	if err != nil {
		return fmt.Errorf("failed to stat source directory: %w", err)
	}

	if err := os.MkdirAll(dst, srcInfo.Mode()); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	entries, err := os.ReadDir(src)
	if err != nil {
		return fmt.Errorf("failed to read source directory: %w", err)
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			if err := hardLinkDir(srcPath, dstPath); err != nil {
				return err
			}
		} else {
			if err := os.Link(srcPath, dstPath); err != nil {
				return fmt.Errorf("failed to create hard link: %w", err)
			}
		}
	}

	return nil
}
