// Package certmanager implements bundle generation for cert-manager.
//
// cert-manager is a cloud-native certificate management solution for Kubernetes,
// automating the issuance and renewal of X.509 certificates from various sources:
//   - Let's Encrypt (ACME protocol)
//   - HashiCorp Vault
//   - Venafi
//   - Self-signed certificates
//   - Private PKI
//
// # Bundle Structure
//
// Generated bundles include:
//   - values.yaml: Helm chart configuration
//   - checksums.txt: SHA256 checksums for verification
//
// # Implementation
//
// This bundler uses the generic bundler framework from [internal.ComponentConfig]
// and [internal.MakeBundle]. The componentConfig variable defines:
//   - Node selector paths for controller, webhook, cainjector, and startupapicheck
//   - Default Helm repository (https://charts.jetstack.io)
//   - Default Helm chart (jetstack/cert-manager)
//
// # Usage
//
// The bundler is registered in the global bundler registry and can be invoked
// via the CLI or programmatically:
//
//	bundler := certmanager.NewBundler(config)
//	result, err := bundler.Make(ctx, recipe, outputDir)
//
// Or through the bundler framework:
//
//	cnsctl bundle --recipe recipe.yaml --bundlers cert-manager --output ./bundles
//
// # Configuration Extraction
//
// The bundler extracts values from recipe component references including
// CRD installation settings, DNS configuration, and webhook settings.
//
// # Prerequisites
//
// Before deploying cert-manager:
//   - Kubernetes 1.22+ cluster
//   - Helm 3.x installed
//   - kubectl configured
//   - Appropriate RBAC permissions
package certmanager
