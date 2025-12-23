/*
Copyright © 2025 NVIDIA Corporation
SPDX-License-Identifier: Apache-2.0
*/
package cli

import (
	"fmt"

	"github.com/NVIDIA/cloud-native-stack/pkg/collectors"
	"github.com/NVIDIA/cloud-native-stack/pkg/serializers"
	"github.com/NVIDIA/cloud-native-stack/pkg/snapshotter"

	"github.com/spf13/cobra"
)

var (
	systemdServices []string
)

// snapshotCmd represents the snapshot command
var snapshotCmd = &cobra.Command{
	Use:     "snapshot",
	Aliases: []string{"snap"},
	GroupID: "functional",
	Short:   "Capture system configuration snapshot",
	Long: `Capture a comprehensive snapshot of system configuration including:
  - Loaded kernel modules
  - SystemD service configurations
  - GRUB boot parameters
  - Sysctl kernel parameters

The snapshot can be output in JSON, YAML, or table format.`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		ctx := cmd.Context()

		// Parse output format
		outFormat := serializers.Format(format)
		if outFormat.IsUnknown() {
			return fmt.Errorf("unknown output format: %q", outFormat)
		}

		// Create factory with configured services
		factory := &collectors.DefaultCollectorFactory{
			SystemDServices: systemdServices,
		}

		// Create and run snapshotter
		ns := snapshotter.NodeSnapshotter{
			Version:    version,
			Factory:    factory,
			Serializer: serializers.NewFileWriterOrStdout(outFormat, output),
		}

		return ns.Run(ctx)
	},
}

func init() {
	rootCmd.AddCommand(snapshotCmd)

	// Define output format flag
	snapshotCmd.Flags().StringVarP(&output, "output", "", "", "output file path (default: stdout)")
	snapshotCmd.Flags().StringVarP(&format, "format", "", "json", "output format (json, yaml, table)")

	// Define systemd services to snapshot
	snapshotCmd.Flags().StringSliceVar(&systemdServices, "systemd-services",
		[]string{"containerd.service", "docker.service", "kubelet.service"},
		"systemd services to snapshot")
}
