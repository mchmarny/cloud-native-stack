// Copyright (c) 2025, NVIDIA CORPORATION.  All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package result provides types for tracking bundler execution results.
//
// This package defines structures for capturing individual bundler results and
// aggregating them into a final output report. It supports both successful
// executions and error tracking.
//
// # Core Types
//
// Result: Individual bundler execution result
//
//	type Result struct {
//	    BundlerType types.BundleType  // Type of bundler executed
//	    Success     bool               // Whether execution succeeded
//	    Files       []string           // Generated file paths
//	    Checksums   map[string]string  // SHA256 checksums
//	    Metadata    map[string]any     // Additional metadata
//	    Duration    time.Duration      // Execution time
//	    Error       error              // Error if failed
//	}
//
// Output: Aggregated results from all bundlers
//
//	type Output struct {
//	    Results      []*Result          // All bundler results
//	    TotalFiles   int                // Total files generated
//	    SuccessCount int                // Number of successful bundlers
//	    FailureCount int                // Number of failed bundlers
//	    TotalTime    time.Duration      // Total execution time
//	}
//
// # Usage - Creating Results
//
// Results are typically created by BaseBundler:
//
//	result := &result.Result{
//	    BundlerType: types.BundleType("gpu-operator"),
//	    Success:     true,
//	    Files:       []string{"values.yaml", "manifests/clusterpolicy.yaml"},
//	    Checksums:   map[string]string{"values.yaml": "sha256..."},
//	    Duration:    2 * time.Second,
//	}
//
// # Usage - Aggregating Results
//
// Output aggregates multiple results with statistics:
//
//	output := &result.Output{
//	    Results: []*result.Result{result1, result2},
//	}
//	output.Compute() // Calculate statistics
//
//	fmt.Printf("Success: %d, Failed: %d\n", output.SuccessCount, output.FailureCount)
//	fmt.Printf("Total files: %d\n", output.TotalFiles)
//	fmt.Printf("Total time: %v\n", output.TotalTime)
//
// # Output Formatting
//
// Summary: Human-readable summary of results
//
//	fmt.Println(output.Summary())
//	// Output:
//	// Generated 2 bundles (2 succeeded, 0 failed) with 15 files in 2.5s
//
// Details: Detailed breakdown of each bundler
//
//	fmt.Println(output.Details())
//	// Output:
//	// gpu-operator: ✓ 8 files in 1.2s
//	// network-operator: ✓ 7 files in 1.3s
//
// # Serialization
//
// Results can be serialized to JSON or YAML:
//
//	data, err := json.MarshalIndent(output, "", "  ")
//	fmt.Println(string(data))
//
// # Error Handling
//
// Failed bundlers include error information:
//
//	if !result.Success {
//	    fmt.Printf("Bundler %s failed: %v\n", result.BundlerType, result.Error)
//	}
//
// Output tracks both successful and failed results:
//
//	for _, r := range output.Results {
//	    if !r.Success {
//	        log.Printf("Failed: %s - %v", r.BundlerType, r.Error)
//	    }
//	}
//
// # Metadata
//
// Results can include additional metadata:
//
//	result.Metadata["recipe_version"] = "v1.0.0"
//	result.Metadata["gpu_operator_version"] = "25.3.1"
//	result.Metadata["bundle_dir"] = "/path/to/bundle"
//
// Metadata is preserved during serialization and can be used for auditing,
// debugging, or integration with external tools.
//
// # Thread Safety
//
// Individual Result instances are not thread-safe. However, Output can safely
// aggregate results from concurrent bundler executions since each Result is
// independent.
package result
