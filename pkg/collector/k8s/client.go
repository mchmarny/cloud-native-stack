package k8s

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

var (
	clientOnce   sync.Once
	cachedClient *kubernetes.Clientset
	cachedConfig *rest.Config
	clientErr    error
)

// GetKubeClient returns a singleton Kubernetes client, creating it on first call.
// Subsequent calls return the cached client for connection reuse and reduced overhead.
// This prevents connection exhaustion and reduces load on the Kubernetes API server.
func GetKubeClient() (*kubernetes.Clientset, *rest.Config, error) {
	clientOnce.Do(func() {
		cachedClient, cachedConfig, clientErr = buildKubeClient("")
	})
	return cachedClient, cachedConfig, clientErr
}

// buildKubeClient creates a Kubernetes client from the given kubeconfig file.
// If kubeconfig is empty, it will look for the KUBECONFIG environment variable
// or the default kubeconfig file in the user's home directory.
// It returns the Kubernetes client and the rest.Config used to create it.
// If an error occurs during the process, it returns nil and the error.
func buildKubeClient(kubeconfig string) (*kubernetes.Clientset, *rest.Config, error) {
	if kubeconfig == "" {
		kubeconfig = os.Getenv("KUBECONFIG")

		if kubeconfig == "" {
			kubeconfig = filepath.Join(homedir.HomeDir(), ".kube", "config")
			if _, err := os.Stat(kubeconfig); os.IsNotExist(err) {
				kubeconfig = ""
			}
		}
	}

	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to build kube config: %w", err)
	}

	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create kubernetes client: %w", err)
	}

	return client, config, nil
}
