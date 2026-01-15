package serializer

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/NVIDIA/cloud-native-stack/pkg/k8s/client"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// version is the application version used in ConfigMap labels.
var version = "unknown"

// SetVersion sets the version used in ConfigMap labels.
func SetVersion(v string) {
	version = v
}

// ConfigMapWriter writes serialized data to a Kubernetes ConfigMap.
// The ConfigMap is created if it doesn't exist, or updated if it does.
type ConfigMapWriter struct {
	namespace string
	name      string
	format    Format
}

// NewConfigMapWriter creates a new ConfigMapWriter that writes to the specified
// namespace and ConfigMap name in the given format.
func NewConfigMapWriter(namespace, name string, format Format) *ConfigMapWriter {
	if format.IsUnknown() {
		slog.Warn("unknown format, defaulting to JSON", "format", format)
		format = FormatJSON
	}
	return &ConfigMapWriter{
		namespace: namespace,
		name:      name,
		format:    format,
	}
}

// Serialize writes the snapshot data to a ConfigMap.
// The ConfigMap will have:
// - data.snapshot.{yaml|json}: The serialized snapshot content
// - data.format: The format used (yaml or json)
// - data.timestamp: ISO 8601 timestamp of when the snapshot was created
func (w *ConfigMapWriter) Serialize(ctx context.Context, snapshot any) error {
	// Create context with timeout for Kubernetes API operations
	// Use longer timeout to accommodate rate limiter after heavy API usage
	writeCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	client, config, err := client.GetKubeClient()
	if err != nil {
		return fmt.Errorf("failed to get kubernetes client: %w", err)
	}

	// Log authentication context for audit
	authInfo := "default"
	switch {
	case config.AuthProvider != nil:
		authInfo = config.AuthProvider.Name
	case config.ExecProvider != nil:
		authInfo = "exec"
	case config.BearerToken != "":
		authInfo = "bearer-token"
	case config.CertData != nil:
		authInfo = "cert"
	}

	slog.Info("configmap operation",
		"namespace", w.namespace,
		"name", w.name,
		"auth_method", authInfo,
		"format", w.format)

	// Serialize snapshot to bytes using appropriate format
	var content []byte
	var extension string
	switch w.format {
	case FormatJSON:
		content, err = serializeJSON(snapshot)
		extension = "json"
	case FormatYAML:
		content, err = serializeYAML(snapshot)
		extension = "yaml"
	case FormatTable:
		content, err = serializeTable(snapshot)
		extension = "txt"
	default:
		return fmt.Errorf("unsupported format for ConfigMap: %s", w.format)
	}
	if err != nil {
		return fmt.Errorf("failed to serialize snapshot: %w", err)
	}

	// Create ConfigMap data
	timestamp := time.Now().UTC().Format(time.RFC3339)
	dataKey := fmt.Sprintf("snapshot.%s", extension)
	configMapData := map[string]string{
		dataKey:     string(content),
		"format":    string(w.format),
		"timestamp": timestamp,
	}

	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      w.name,
			Namespace: w.namespace,
			Labels: map[string]string{
				"app.kubernetes.io/name":      "cns",
				"app.kubernetes.io/component": "snapshot",
				"app.kubernetes.io/version":   version,
			},
		},
		Data: configMapData,
	}

	cmClient := client.CoreV1().ConfigMaps(w.namespace)

	// Try to get existing ConfigMap
	existing, err := cmClient.Get(writeCtx, w.name, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			// Create new ConfigMap
			slog.Info("creating ConfigMap",
				"namespace", w.namespace,
				"name", w.name,
				"format", w.format)
			_, err = cmClient.Create(writeCtx, configMap, metav1.CreateOptions{})
			if err != nil {
				return fmt.Errorf("failed to create ConfigMap: %w", err)
			}
			return nil
		}
		return fmt.Errorf("failed to get ConfigMap: %w", err)
	}

	// Update existing ConfigMap
	existing.Data = configMapData
	existing.Labels = configMap.Labels
	slog.Info("updating ConfigMap",
		"namespace", w.namespace,
		"name", w.name,
		"format", w.format)
	_, err = cmClient.Update(writeCtx, existing, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update ConfigMap: %w", err)
	}

	return nil
}

// Close is a no-op for ConfigMapWriter as there are no resources to release.
// This method exists to satisfy the Closer interface.
func (w *ConfigMapWriter) Close() error {
	return nil
}

// parseConfigMapURI parses a ConfigMap URI in the format cm://namespace/name
// and returns the namespace and name components.
// Returns an error if the URI is malformed.
func parseConfigMapURI(uri string) (namespace, name string, err error) {
	if !strings.HasPrefix(uri, ConfigMapURIScheme) {
		return "", "", fmt.Errorf("invalid ConfigMap URI: must start with %s", ConfigMapURIScheme)
	}

	// Remove cm:// prefix
	path := strings.TrimPrefix(uri, ConfigMapURIScheme)

	// Split into namespace/name
	parts := strings.SplitN(path, "/", 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid ConfigMap URI format: expected %snamespace/name, got %s", ConfigMapURIScheme, uri)
	}

	namespace = strings.TrimSpace(parts[0])
	name = strings.TrimSpace(parts[1])

	if namespace == "" {
		return "", "", fmt.Errorf("invalid ConfigMap URI: namespace cannot be empty")
	}
	if name == "" {
		return "", "", fmt.Errorf("invalid ConfigMap URI: name cannot be empty")
	}

	return namespace, name, nil
}
