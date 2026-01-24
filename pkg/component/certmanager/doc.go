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
//   - README.md: Deployment documentation with prerequisites and instructions
//   - checksums.txt: SHA256 checksums for verification
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
// The bundler extracts configuration from recipe measurements:
//   - K8s image subtype: cert-manager version
//   - K8s config subtype: CRD installation, DNS settings, webhooks
//
// # Templates
//
// Templates are embedded in the binary using go:embed and rendered with Go's
// text/template package. Templates support:
//   - Conditional sections based on enabled features
//   - Version-specific configurations
//   - Namespace customization
//   - Resource quota and limit configurations
//
// # Prerequisites
//
// Before deploying cert-manager:
//   - Kubernetes 1.22+ cluster
//   - Helm 3.x installed
//   - kubectl configured
//   - Appropriate RBAC permissions
//
// For ACME/Let's Encrypt:
//   - External DNS provider access (Route53, CloudFlare, etc.)
//   - Valid domain name for DNS-01 challenge
//
// # Security Considerations
//
// cert-manager requires cluster-wide permissions for certificate management:
//   - ClusterRole for CRD access
//   - Webhooks for validation and mutation
//   - Secrets for certificate storage
//
// Enable RBAC and network policies to restrict access appropriately.
package certmanager
