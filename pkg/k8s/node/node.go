package node

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/NVIDIA/cloud-native-stack/pkg/k8s/client"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8s "k8s.io/client-go/kubernetes"
)

// ListOptions contains the configuration options for listing nodes in a Kubernetes cluster.
type ListOptions struct {
	// Kubeconfig is the path to the kubeconfig file.
	Kubeconfig string
	// LabelSelector is a selector to filter nodes based on labels.
	LabelSelector string
	// FieldSelector is a selector to filter nodes based on fields.
	FieldSelector string
	// Limit is the maximum number of nodes to return (default is 100).
	Limit int64
	// client is the Kubernetes client to use for testing.
	Client k8s.Interface
}

type Node struct {
	// Name is the name of the node.
	Name string `json:"name" yaml:"name"`
	// Role is the role of the node, derived from its labels.
	Role string `json:"role" yaml:"role"`
	// Age is the age of the node as a duration since its creation.
	Age string `json:"age" yaml:"age"`
	// Host is the provider hostname of the node.
	Host string `json:"host,omitempty" yaml:"host,omitempty"`
	// IP is the primary IP address of the node.
	IP string `json:"ip,omitempty" yaml:"ip,omitempty"`
}

// Summary returns a list of all nodes in the cluster.
// It uses the List function to get the list of nodes and then extracts
// the relevant information (name, role, and age) for each node.
func Summary(ctx context.Context, kubeconfig string) ([]*Node, error) {
	list, err := List(ctx, ListOptions{
		Kubeconfig: kubeconfig,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list nodes: %w", err)
	}

	nodes := make([]*Node, 0, len(list))
	for _, n := range list {
		node := &Node{
			Name: n.Name,
			Role: ParseNodeRole(n),
			Age:  FormatAge(n.CreationTimestamp.Time),
			IP:   getNodeIP(n, v1.NodeInternalIP),
		}

		if node.Host, err = parseHostID(n.Spec.ProviderID); err != nil {
			slog.Warn("failed to parse node providerID", "node", n.Name, "providerID", n.Spec.ProviderID, "error", err)
			node.Host = n.Spec.ProviderID
		}

		nodes = append(nodes, node)
	}

	sort.Slice(nodes, func(i, j int) bool {
		return strings.ToLower(nodes[i].Name) < strings.ToLower(nodes[j].Name)
	})

	return nodes, nil
}

const (
	MinuteDuration = time.Minute
	HourDuration   = time.Hour
	DayDuration    = 24 * HourDuration
)

// FormatAge formats the age of a node as a human-readable string.
// It calculates the duration since the node was created and formats it
// into a string that includes days, hours, and minutes.
// If the node was created less than a minute ago, it returns "0m".
func FormatAge(createdOn time.Time) string {
	d := metav1.Now().Sub(createdOn)

	if d < MinuteDuration {
		return "0m"
	}

	days := d / DayDuration
	d -= days * DayDuration

	hours := d / HourDuration
	d -= hours * HourDuration

	minutes := d / MinuteDuration

	var parts []string
	if days > 0 {
		parts = append(parts, fmt.Sprintf("%d days", days))
	}
	if hours > 0 {
		parts = append(parts, fmt.Sprintf("%d hours", hours))
	}
	if minutes > 0 || len(parts) == 0 {
		parts = append(parts, fmt.Sprintf("%d minutes", minutes))
	}

	return fmt.Sprint(parts[0], func() string {
		if len(parts) > 1 {
			return " " + parts[1]
		}
		return ""
	}())
}

const (
	nodeListPageSizeDefault int64 = 500
	nodeListAbsoluteMax     int64 = 10000 // Hard cap to prevent memory exhaustion
)

// List returns a list of nodes in the cluster based on the provided options.
// It uses the Kubernetes client to fetch the nodes and applies the specified
// label and field selectors. The function implements pagination to handle large
// clusters with thousands of nodes efficiently. If no limit is specified per page,
// it defaults to 500 nodes per page to balance performance and memory usage.
func List(ctx context.Context, opt ListOptions) ([]*v1.Node, error) {
	if opt.Client == nil {
		client, _, err := client.GetKubeClient(opt.Kubeconfig)
		if err != nil {
			return nil, fmt.Errorf("failed to get kubernetes client: %w", err)
		}
		opt.Client = client
	}

	// Enforce absolute maximum to prevent memory exhaustion
	effectiveLimit := opt.Limit
	if effectiveLimit == 0 || effectiveLimit > nodeListAbsoluteMax {
		effectiveLimit = nodeListAbsoluteMax
	}

	pageSize := nodeListPageSizeDefault
	if effectiveLimit > 0 && effectiveLimit < pageSize {
		pageSize = effectiveLimit
	}

	// Pre-allocate with reasonable capacity to prevent repeated slice growth
	estimatedCap := effectiveLimit
	if estimatedCap > 500 {
		estimatedCap = 500
	}
	var allNodes []*v1.Node
	if estimatedCap > 0 {
		allNodes = make([]*v1.Node, 0, estimatedCap)
	}
	continueToken := ""
	totalFetched := int64(0)

	for {
		currentLimit := pageSize
		if effectiveLimit > 0 && (totalFetched+currentLimit) > effectiveLimit {
			currentLimit = effectiveLimit - totalFetched
		}

		lo := metav1.ListOptions{
			LabelSelector: opt.LabelSelector,
			FieldSelector: opt.FieldSelector,
			Limit:         currentLimit,
			Continue:      continueToken,
		}

		slog.Debug("fetching nodes",
			slog.Int64("limit", currentLimit),
			slog.Int64("totalSoFar", totalFetched),
			slog.Bool("hasContinueToken", continueToken != ""),
		)

		list, err := opt.Client.CoreV1().Nodes().List(ctx, lo)
		if err != nil {
			return nil, fmt.Errorf("failed to get nodes: %w", err)
		}

		pageNodes := make([]*v1.Node, 0, len(list.Items))
		for i := range list.Items {
			pageNodes = append(pageNodes, &list.Items[i])
		}

		allNodes = append(allNodes, pageNodes...)
		totalFetched += int64(len(pageNodes))

		slog.Debug("fetched nodes page",
			slog.Int("pageSize", len(pageNodes)),
			slog.Int64("totalFetched", totalFetched),
			slog.Bool("hasMore", list.Continue != ""),
		)

		continueToken = list.Continue
		if continueToken == "" {
			break
		}

		if effectiveLimit > 0 && totalFetched >= effectiveLimit {
			break
		}

		if len(pageNodes) == 0 {
			slog.Warn("received empty page with continue token, stopping pagination")
			break
		}
	}

	slog.Debug("node list complete",
		slog.Int("totalNodes", len(allNodes)),
		slog.Int64("requestedLimit", opt.Limit),
	)

	return allNodes, nil
}

// ListOptions contains the configuration options for listing nodes in a Kubernetes cluster.
type GetOptions struct {
	// Kubeconfig is the path to the kubeconfig file.
	Kubeconfig string
	// Name is the name of the node to get.
	Name string

	// client is the Kubernetes client to use for testing.
	client k8s.Interface
}

// Validate checks if the GetOptions are valid.
func (opt *GetOptions) Validate() error {
	if opt.Name == "" {
		return fmt.Errorf("node name is required")
	}
	return nil
}

// Get returns a specific node in the cluster based on the provided options.
// It uses the Kubernetes client to fetch the node. If no node name is provided,
// it attempts to detect the current node from the NODE_NAME environment variable
// (typically set via Kubernetes Downward API in pod specs).
// If the node is not found, it returns a NotFound error.
func Get(ctx context.Context, opt GetOptions) (*v1.Node, error) {
	// If no node name provided, try to get current node from environment
	if opt.Name == "" {
		nodeName := getNodeName()
		if nodeName == "" {
			return nil, fmt.Errorf("node name not provided and could not detect current node (set NODE_NAME environment variable)")
		}
		opt.Name = nodeName
		slog.Debug("using current node from environment", "node", nodeName)
	}

	if err := opt.Validate(); err != nil {
		return nil, fmt.Errorf("invalid options: %w", err)
	}

	if opt.client == nil {
		client, _, err := client.GetKubeClient(opt.Kubeconfig)
		if err != nil {
			return nil, fmt.Errorf("failed to get kubernetes client: %w", err)
		}
		opt.client = client
	}

	node, err := opt.client.CoreV1().Nodes().Get(ctx, opt.Name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get node: %w", err)
	}

	return node, nil
}

const (
	NodeRoleLabelPrefix = "node-role.kubernetes.io/"
	NodeRoleLabel       = "nodeRole"
	NodeRoleUndefined   = "undefined"
)

// ParseNodeRole parses the node role from the node labels.
// It looks for labels that start with the NodeRoleLabelPrefix and returns
// the role name. If no role is found, it returns NodeRoleUndefined.
// This function is useful for determining the role of a node in a Kubernetes cluster.
// It is used to identify the role of a node based on its labels.
// The function iterates through the labels of the node and checks for labels
// that start with the NodeRoleLabelPrefix. If a matching label is found, it
// extracts the role name by trimming the prefix. If no matching label is found,
// it returns NodeRoleUndefined.
func ParseNodeRole(n *v1.Node) string {
	// try actual node role labels first
	for k := range n.Labels {
		if strings.HasPrefix(k, NodeRoleLabelPrefix) {
			role := strings.TrimPrefix(k, NodeRoleLabelPrefix)
			if role != "" {
				return role
			}
		}
	}

	// if that fails, try the NV common `nodeRole` label
	for k, v := range n.Labels {
		if strings.EqualFold(k, NodeRoleLabel) {
			return v
		}
	}

	return NodeRoleUndefined
}

func parseHostID(providerID string) (string, error) {
	if providerID == "" {
		return "", fmt.Errorf("node providerID is empty")
	}

	parts := strings.Split(providerID, ":")
	if len(parts) < 2 {
		return "", fmt.Errorf("invalid providerID format: %s", providerID)
	}

	provider := strings.ToLower(parts[0])
	details := strings.TrimPrefix(providerID, provider+"://")

	subParts := strings.Split(details, "/")
	if len(subParts) == 0 {
		return "", fmt.Errorf("invalid providerID details format: %s", details)
	}

	provider = subParts[len(subParts)-1]

	return provider, nil
}

// getNodeIP retrieves the IP address of the node for the specified address type.
func getNodeIP(node *v1.Node, ipType v1.NodeAddressType) string {
	for _, addr := range node.Status.Addresses {
		if addr.Type == ipType {
			return addr.Address
		}
	}
	return ""
}

// getNodeName retrieves the current node name from environment variables.
// It checks NODE_NAME first (typically set via Downward API), then falls back
// to KUBERNETES_NODE_NAME, and finally HOSTNAME as a last resort.
func getNodeName() string {
	// Preferred: NODE_NAME set via Downward API
	if nodeName := os.Getenv("NODE_NAME"); nodeName != "" {
		return nodeName
	}

	// Alternative: KUBERNETES_NODE_NAME
	if nodeName := os.Getenv("KUBERNETES_NODE_NAME"); nodeName != "" {
		return nodeName
	}

	// Last resort: HOSTNAME (may be pod name, not node name)
	// This is less reliable but included for compatibility
	if hostname := os.Getenv("HOSTNAME"); hostname != "" {
		return hostname
	}

	return ""
}
