/*
Copyright © 2025 NVIDIA Corporation
SPDX-License-Identifier: Apache-2.0
*/
package cli

import (
	"context"
	"fmt"
	"time"

	"github.com/urfave/cli/v3"

	"github.com/NVIDIA/cloud-native-stack/pkg/collector"
	"github.com/NVIDIA/cloud-native-stack/pkg/serializer"
	"github.com/NVIDIA/cloud-native-stack/pkg/snapshotter"
)

func snapshotCmd() *cli.Command {
	return &cli.Command{
		Name:                  "snapshot",
		EnableShellCompletion: true,
		Usage:                 "Capture system configuration snapshot",
		Description: `Capture a comprehensive snapshot of system configuration including:
  - CPU and GPU settings
  - GRUB boot parameters
  - Kubernetes cluster configuration
  - Loaded kernel modules
  - Sysctl kernel parameters
  - SystemD service configurations

The snapshot can be output in JSON, YAML, or table format.

# Agent Deployment Mode

Use --deploy-agent to deploy a Kubernetes Job that captures the snapshot on GPU nodes:

  cnsctl snapshot --deploy-agent --namespace gpu-operator --output cm://gpu-operator/cns-snapshot

The agent mode will:
  1. Deploy RBAC resources (ServiceAccount, Role, RoleBinding, ClusterRole, ClusterRoleBinding)
  2. Deploy a Job on GPU nodes to capture the snapshot
  3. Wait for the Job to complete
  4. Retrieve the snapshot from the ConfigMap
  5. Save to file or stdout
  6. Clean up the Job (optionally keep RBAC for reuse)

# Examples

Basic agent deployment:
  cnsctl snapshot --deploy-agent

Target specific GPU nodes with node selector:
  cnsctl snapshot --deploy-agent --node-selector nodeGroup=customer-gpu

Override default tolerations (by default, all taints are tolerated):
  cnsctl snapshot --deploy-agent \
    --toleration dedicated=user-workload:NoSchedule

Combined node selector and custom tolerations:
  cnsctl snapshot --deploy-agent \
    --node-selector nodeGroup=customer-gpu \
    --toleration dedicated=user-workload:NoSchedule \
    --output cm://gpu-operator/cns-snapshot`,
		Flags: []cli.Flag{
			// Agent deployment flags
			&cli.BoolFlag{
				Name:  "deploy-agent",
				Usage: "Deploy Kubernetes Job to capture snapshot on GPU nodes",
			},
			&cli.StringFlag{
				Name:    "namespace",
				Usage:   "Kubernetes namespace for agent deployment",
				Sources: cli.EnvVars("CNS_NAMESPACE"),
				Value:   "gpu-operator",
			},
			&cli.StringFlag{
				Name:    "image",
				Usage:   "Container image for agent Job",
				Sources: cli.EnvVars("CNS_IMAGE"),
				Value:   "ghcr.io/nvidia/cns:latest",
			},
			&cli.StringSliceFlag{
				Name:  "image-pull-secret",
				Usage: "Secret name for pulling images from private registries (can be repeated)",
			},
			&cli.StringFlag{
				Name:  "job-name",
				Usage: "Override default Job name",
				Value: "cns",
			},
			&cli.StringFlag{
				Name:  "service-account-name",
				Usage: "Override default ServiceAccount name",
				Value: "cns",
			},
			&cli.StringSliceFlag{
				Name:  "node-selector",
				Usage: "Node selector for Job scheduling (format: key=value, can be repeated)",
			},
			&cli.StringSliceFlag{
				Name:  "toleration",
				Usage: "Toleration for Job scheduling (format: key=value:effect). By default, all taints are tolerated. Specifying this flag overrides the defaults.",
			},
			&cli.DurationFlag{
				Name:  "timeout",
				Usage: "Timeout for waiting for Job completion",
				Value: 5 * time.Minute,
			},
			&cli.BoolFlag{
				Name:  "cleanup",
				Value: true,
				Usage: "Remove Job and RBAC resources on completion",
			},
			outputFlag,
			formatFlag,
			kubeconfigFlag,
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			// Parse output format
			outFormat := serializer.Format(cmd.String("format"))
			if outFormat.IsUnknown() {
				return fmt.Errorf("unknown output format: %q", outFormat)
			}

			// Create factory
			factory := collector.NewDefaultFactory(
				collector.WithVersion(version),
			)

			// Build snapshotter configuration
			ns := snapshotter.NodeSnapshotter{
				Version:    version,
				Factory:    factory,
				Serializer: serializer.NewFileWriterOrStdout(outFormat, cmd.String("output")),
			}

			// Check if agent deployment mode is enabled
			if cmd.Bool("deploy-agent") {
				// Parse node selectors
				nodeSelector, err := snapshotter.ParseNodeSelectors(cmd.StringSlice("node-selector"))
				if err != nil {
					return fmt.Errorf("invalid node-selector: %w", err)
				}

				// Parse tolerations
				tolerations, err := snapshotter.ParseTolerations(cmd.StringSlice("toleration"))
				if err != nil {
					return fmt.Errorf("invalid toleration: %w", err)
				}

				// Configure agent deployment
				ns.AgentConfig = &snapshotter.AgentConfig{
					Enabled:            true,
					Kubeconfig:         cmd.String("kubeconfig"),
					Namespace:          cmd.String("namespace"),
					Image:              cmd.String("image"),
					ImagePullSecrets:   cmd.StringSlice("image-pull-secret"),
					JobName:            cmd.String("job-name"),
					ServiceAccountName: cmd.String("service-account-name"),
					NodeSelector:       nodeSelector,
					Tolerations:        tolerations,
					Timeout:            cmd.Duration("timeout"),
					Cleanup:            cmd.Bool("cleanup"),
					Output:             cmd.String("output"),
					Debug:              cmd.Bool("debug"),
				}
			}

			// Execute snapshot (routes to local or agent based on config)
			return ns.Measure(ctx)
		},
	}
}
