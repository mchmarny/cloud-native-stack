// Package oci provides functionality for packaging and pushing artifacts to OCI-compliant registries.
//
// This package enables bundled artifacts to be pushed to any OCI-compliant registry
// (Docker Hub, GHCR, ECR, local registries, etc.) using the ORAS (OCI Registry As Storage) library.
// Artifacts are packaged as OCI Image Layout format and can be pushed to remote registries.
//
// # Overview
//
// The package provides two main operations:
//   - Package: Creates a local OCI artifact in OCI Image Layout format
//   - PushFromStore: Pushes a previously packaged artifact to a remote registry
//
// These can be combined for a package-then-push workflow, or used independently.
//
// # Core Types
//
//   - PackageOptions: Configuration for local OCI packaging
//   - PackageResult: Result of local packaging (digest, reference, store path)
//   - PushOptions: Configuration for pushing to remote registries
//   - PushResult: Result of a successful push (digest, reference)
//
// # Usage
//
// Package and push in two steps:
//
//	// First, package locally
//	pkgResult, err := oci.Package(ctx, oci.PackageOptions{
//	    SourceDir:  "/path/to/bundle",
//	    OutputDir:  "/path/to/output",
//	    Registry:   "ghcr.io",
//	    Repository: "nvidia/bundle",
//	    Tag:        "v1.0.0",
//	})
//	if err != nil {
//	    return err
//	}
//
//	// Then push to registry
//	pushResult, err := oci.PushFromStore(ctx, pkgResult.StorePath, oci.PushOptions{
//	    Registry:   "ghcr.io",
//	    Repository: "nvidia/bundle",
//	    Tag:        "v1.0.0",
//	})
//
// # Configuration
//
// PackageOptions supports configuration for reproducible builds:
//   - ReproducibleTimestamp: Set a fixed timestamp for reproducible builds
//
// PushOptions supports several configuration options:
//   - PlainHTTP: Use HTTP instead of HTTPS (for local development registries)
//   - InsecureTLS: Skip TLS certificate verification
//
// # Authentication
//
// The package automatically uses Docker credential helpers for authentication.
// Credentials are loaded from the standard Docker configuration (~/.docker/config.json)
// using the ORAS credentials package.
//
// # Artifact Type
//
// Artifacts are pushed with the media type "application/vnd.nvidia.cns.artifact".
// This custom media type identifies CNS bundles and distinguishes them from
// runnable container images. Consumers that don't understand this type should
// treat the artifact as a non-executable blob.
package oci
