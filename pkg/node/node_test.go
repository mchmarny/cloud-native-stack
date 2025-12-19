package node

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sfake "k8s.io/client-go/kubernetes/fake"
)

func TestFormatAge(t *testing.T) {
	now := metav1.Now()

	tests := []struct {
		name     string
		created  time.Time
		expected string
	}{
		{
			name:     "Less than a minute",
			created:  now.Add(-30 * time.Second),
			expected: "0m",
		},
		{
			name:     "Exactly one minute",
			created:  now.Add(-1 * time.Minute),
			expected: "1 minutes",
		},
		{
			name:     "59 minutes",
			created:  now.Add(-59 * time.Minute),
			expected: "59 minutes",
		},
		{
			name:     "1 hour",
			created:  now.Add(-1 * time.Hour),
			expected: "1 hours",
		},
		{
			name:     "1 hour 30 minutes",
			created:  now.Add(-1*time.Hour - 30*time.Minute),
			expected: "1 hours 30 minutes",
		},
		{
			name:     "1 day",
			created:  now.Add(-24 * time.Hour),
			expected: "1 days",
		},
		{
			name:     "1 day 2 hours",
			created:  now.Add(-26 * time.Hour),
			expected: "1 days 2 hours",
		},
		{
			name:     "1 day 0 hours 5 minutes",
			created:  now.Add(-24*time.Hour - 5*time.Minute),
			expected: "1 days 5 minutes",
		},
		{
			name:     "2 days 3 hours",
			created:  now.Add(-2*24*time.Hour - 3*time.Hour),
			expected: "2 days 3 hours",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatAge(tt.created)
			if got != tt.expected {
				t.Errorf("FormatAge() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestListNodes(t *testing.T) {
	ctx := context.TODO()

	//nolint:staticcheck // SA1019: NewSimpleClientset is adequate for basic test needs
	fakeClient := k8sfake.NewSimpleClientset(
		&v1.Node{
			ObjectMeta: metav1.ObjectMeta{
				Name:   "node-1",
				Labels: map[string]string{"zone": "us-west"},
			},
		},
		&v1.Node{
			ObjectMeta: metav1.ObjectMeta{
				Name:   "node-2",
				Labels: map[string]string{"zone": "us-east"},
			},
		},
	)

	opts := ListOptions{
		Client: fakeClient,
	}

	nodes, err := List(ctx, opts)
	assert.NoError(t, err)
	assert.Len(t, nodes, 2)
	assert.Equal(t, "node-1", nodes[0].Name)
	assert.Equal(t, "node-2", nodes[1].Name)
}

func TestGetNode(t *testing.T) {
	//nolint:staticcheck // SA1019: NewSimpleClientset is adequate for basic test needs
	fakeClient := k8sfake.NewSimpleClientset(
		&v1.Node{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-node",
			},
		},
	)

	opts := GetOptions{
		Name:   "test-node",
		client: fakeClient,
	}

	n, err := Get(opts)
	assert.NoError(t, err)
	assert.NotNil(t, n)
	assert.Equal(t, "test-node", n.Name)
}

func TestParseNodeRole(t *testing.T) {
	tests := []struct {
		name     string
		node     *v1.Node
		expected string
	}{
		{
			name: "node with worker role",
			node: &v1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"node-role.kubernetes.io/worker": "",
					},
				},
			},
			expected: "worker",
		},
		{
			name: "node with empty role label",
			node: &v1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"node-role.kubernetes.io/": "",
					},
				},
			},
			expected: NodeRoleUndefined,
		},
		{
			name: "node with no role labels",
			node: &v1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"kubernetes.io/hostname": "node1",
					},
				},
			},
			expected: NodeRoleUndefined,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := ParseNodeRole(tt.node)
			assert.Equal(t, tt.expected, actual)
		})
	}
}
