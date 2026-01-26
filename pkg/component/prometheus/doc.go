// Package prometheus implements bundle generation for Prometheus monitoring.
//
// Prometheus is an open-source monitoring and alerting toolkit designed for
// reliability and scalability in cloud-native environments. It provides:
//   - Time series database for metrics storage
//   - Powerful query language (PromQL) for data analysis
//   - Alerting and notification capabilities via Alertmanager
//   - Service discovery for dynamic environments
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
//   - Default Helm repository (https://prometheus-community.github.io/helm-charts)
//   - Default Helm chart (prometheus-community/prometheus)
//   - Value override key mapping (prometheus)
//
// # Usage
//
// The bundler is registered in the global bundler registry and can be invoked
// via the CLI or programmatically:
//
//	bundler := prometheus.NewBundler(config)
//	result, err := bundler.Make(ctx, recipe, outputDir)
//
// Or through the bundler framework:
//
//	cnsctl bundle --recipe recipe.yaml --bundlers prometheus --output ./bundles
//
// # Configuration Extraction
//
// The bundler extracts values from recipe component references. Common
// configurations include server settings, alertmanager setup, and scrape configs.
package prometheus