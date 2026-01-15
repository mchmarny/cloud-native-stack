package agent

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
)

// clusterRoleName is the name used for the ClusterRole and ClusterRoleBinding.
const clusterRoleName = "cns-node-reader"

// Config holds the configuration for deploying the agent.
type Config struct {
	Namespace          string
	ServiceAccountName string
	JobName            string
	Image              string
	ImagePullSecrets   []string
	NodeSelector       map[string]string
	Tolerations        []corev1.Toleration
	Output             string
	Debug              bool
}

// Deployer manages the deployment and lifecycle of the agent Job.
type Deployer struct {
	clientset kubernetes.Interface
	config    Config
}

// NewDeployer creates a new agent Deployer with the given configuration.
func NewDeployer(clientset kubernetes.Interface, config Config) *Deployer {
	return &Deployer{
		clientset: clientset,
		config:    config,
	}
}

// CleanupOptions controls what resources to remove during cleanup.
type CleanupOptions struct {
	Enabled bool // If true, removes Job and all RBAC resources
}
