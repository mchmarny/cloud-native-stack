/*
Copyright © 2025 NVIDIA Corporation
SPDX-License-Identifier: Apache-2.0
*/
package cli

import (
	"fmt"
	"sync"

	"github.com/NVIDIA/cloud-native-stack/pkg/recommendation"
	"github.com/NVIDIA/cloud-native-stack/pkg/serializers"
	ver "github.com/NVIDIA/cloud-native-stack/pkg/version"

	"github.com/spf13/cobra"
)

var (
	// Flags for recommendation query parameters
	recOs        string
	recOsVersion string
	recKernel    string
	recService   string
	recK8s       string
	recGPU       string
	recIntent    string

	mu sync.RWMutex
)

// recommendCmd represents the recommend command
var recommendCmd = &cobra.Command{
	Use:     "recommend",
	Aliases: []string{"rec"},
	GroupID: "functional",
	Short:   "Generate system recommendations for a given environment",
	Long: `Generate system recommendations based on specified environment parameters including:
  - Operating system and version
  - Kernel version
  - Managed service context
  - Kubernetes cluster version
  - GPU type
  - Workload intent

The recommendation can be output in JSON, YAML, or table format.`,
	RunE: func(_ *cobra.Command, _ []string) error {
		// Parse output format
		outFormat := serializers.Format(format)
		if outFormat.IsUnknown() {
			return fmt.Errorf("unknown output format: %q", outFormat)
		}

		q, err := buildQueryFromFlags()
		if err != nil {
			return fmt.Errorf("error parsing recommendation input parameter: %w", err)
		}

		rec, err := recommendation.BuildRecommendation(q)
		if err != nil {
			return fmt.Errorf("error building recommendation: %w", err)
		}

		return serializers.NewFileWriterOrStdout(outFormat, output).Serialize(rec)
	},
}

func init() {
	rootCmd.AddCommand(recommendCmd)

	// Define flags for recommendation query parameters
	recommendCmd.Flags().StringVarP(&recOs, "os", "", "", "Operating system family (e.g., ubuntu, cos)")
	recommendCmd.Flags().StringVarP(&recOsVersion, "osv", "", "", "Operating system version (e.g., 22.04)")
	recommendCmd.Flags().StringVarP(&recKernel, "kernel", "", "", "Running kernel version (e.g., 5.15.0)")
	recommendCmd.Flags().StringVarP(&recService, "service", "", "", "Managed service context (e.g., eks, gke, or self-managed)")
	recommendCmd.Flags().StringVarP(&recK8s, "k8s", "", "", "Kubernetes cluster version (e.g., v1.25.4)")
	recommendCmd.Flags().StringVarP(&recGPU, "gpu", "", "", "GPU type (e.g., H100, GB200)")
	recommendCmd.Flags().StringVarP(&recIntent, "intent", "", "", "Workload intent (e.g., training or inference)")

	// Define output format flag specific to recommend command
	recommendCmd.Flags().StringVarP(&output, "output", "", "", "output file path (default: stdout)")
	recommendCmd.Flags().StringVarP(&format, "format", "", "json", "output format (json, yaml, table)")
}

// buildQueryFromFlags constructs a recommendation.Query from CLI flags.
func buildQueryFromFlags() (*recommendation.Query, error) {
	mu.Lock()
	defer mu.Unlock()

	q := &recommendation.Query{}
	var err error

	if recOs != "" {
		q.Os = recommendation.OsFamily(recOs)
		if !q.Os.IsValid() {
			return nil, fmt.Errorf("os: %q, supported values: %v", recOs, recommendation.SupportedOSFamilies())
		}
	}
	if recOsVersion != "" {
		q.OsVersion, err = ver.ParseVersion(recOsVersion)
		if err != nil {
			return nil, fmt.Errorf("osv: %q: %w", recOsVersion, err)
		}
	}
	if recKernel != "" {
		q.Kernel, err = ver.ParseVersion(recKernel)
		if err != nil {
			return nil, fmt.Errorf("kernel: %q: %w", recKernel, err)
		}
	}
	if recService != "" {
		q.Service = recommendation.ServiceType(recService)
		if !q.Service.IsValid() {
			return nil, fmt.Errorf("service: %q, supported values: %v", recService, recommendation.SupportedServiceTypes())
		}
	}

	if recK8s != "" {
		q.K8s, err = ver.ParseVersion(recK8s)
		if err != nil {
			return nil, fmt.Errorf("k8s: %q: %w", recK8s, err)
		}
	}
	if recGPU != "" {
		q.GPU = recommendation.GPUType(recGPU)
		if !q.GPU.IsValid() {
			return nil, fmt.Errorf("gpu: %q, supported values: %v", recGPU, recommendation.SupportedGPUTypes())
		}
	}
	if recIntent != "" {
		q.Intent = recommendation.IntentType(recIntent)
		if !q.Intent.IsValid() {
			return nil, fmt.Errorf("intent: %q, supported values: %v", recIntent, recommendation.SupportedIntentTypes())
		}
	}

	return q, nil
}
