// Package prometheus implements bundle generation for Prometheus monitoring.
//
// Prometheus is an open-source monitoring and alerting toolkit designed for
// reliability and scalability. It provides:
//   - Time series database for metrics storage
//   - Powerful query language (PromQL)
//   - Alerting and notification capabilities
//   - Service discovery and metrics collection
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
//   - Node selector and toleration paths for system and accelerated nodes
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
// # Configuration
//
// The bundler extracts values from recipe component references. Common
// configurations include:
//   - Server configuration (retention, storage, resources)
//   - Alertmanager settings for notifications
//   - Service discovery configuration
//   - Scrape configurations for targets
//   - Node exporter settings for host metrics
//
// # Node Scheduling
//
// The bundler supports node selector and toleration configuration for:
//   - System components: server, alertmanager, pushgateway
//   - Accelerated components: nodeExporter (for GPU node monitoring)
package prometheus