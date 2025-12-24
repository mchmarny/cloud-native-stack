/*
Copyright © 2025 NVIDIA Corporation
SPDX-License-Identifier: Apache-2.0
*/
package cli

import (
	"context"
	"fmt"

	"github.com/urfave/cli/v3"

	"github.com/NVIDIA/cloud-native-stack/pkg/collector"
	"github.com/NVIDIA/cloud-native-stack/pkg/serializer"
	"github.com/NVIDIA/cloud-native-stack/pkg/snapshotter"
)

func snapshotCmd() *cli.Command {
	return &cli.Command{
		Name:                  "snapshot",
		Aliases:               []string{"snap"},
		EnableShellCompletion: true,
		Usage:                 "Capture system configuration snapshot",
		Description: `Capture a comprehensive snapshot of system configuration including:
  - CPU and GPU settings
  - GRUB boot parameters
  - Kubernetes cluster configuration
  - Loaded kernel modules
  - Sysctl kernel parameters
  - SystemD service configurations

The snapshot can be output in JSON, YAML, or table format.`,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "output",
				Usage: "output file path (default: stdout)",
			},
			&cli.StringFlag{
				Name:  "format",
				Value: "json",
				Usage: "output format (json, yaml, table)",
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			// Parse output format
			outFormat := serializer.Format(cmd.String("format"))
			if outFormat.IsUnknown() {
				return fmt.Errorf("unknown output format: %q", outFormat)
			}

			// Create factory with configured services
			factory := collector.NewDefaultFactory()

			// Create and run snapshotter
			ns := snapshotter.NodeSnapshotter{
				Version:    version,
				Factory:    factory,
				Serializer: serializer.NewFileWriterOrStdout(outFormat, cmd.String("output")),
			}

			return ns.Run(ctx)
		},
	}
}
