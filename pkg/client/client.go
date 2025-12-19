package client

import (
	"fmt"
	"os"
	"path/filepath"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

// GetKubeClient creates a Kubernetes client from the given kubeconfig file.
// If kubeconfig is empty, it will look for the KUBECONFIG environment variable
// or the default kubeconfig file in the user's home directory.
// It returns the Kubernetes client and the rest.Config used to create it.
// If an error occurs during the process, it returns nil and the error.
func GetKubeClient(kubeconfig string) (*kubernetes.Clientset, *rest.Config, error) {
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
		return nil, nil, fmt.Errorf("failed to create kuebrnetes client: %w", err)
	}

	return client, config, nil
}
