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

package agent

import (
	"context"
	"fmt"
	"strings"

	authv1 "k8s.io/api/authorization/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// PermissionCheck represents a single permission check result.
type PermissionCheck struct {
	Resource  string
	Verb      string
	Namespace string
	Allowed   bool
	Reason    string
}

// CheckPermissions verifies if the current user has the required permissions
// to deploy the agent. Returns a list of permission checks and an error if any
// required permissions are missing.
func (d *Deployer) CheckPermissions(ctx context.Context) ([]PermissionCheck, error) {
	checks := []PermissionCheck{}

	// Required permissions for deployment
	requiredChecks := []struct {
		resource  string
		verb      string
		namespace string
	}{
		// Namespace-scoped resources
		{"serviceaccounts", "create", d.config.Namespace},
		{"roles", "create", d.config.Namespace},
		{"rolebindings", "create", d.config.Namespace},
		{"jobs", "create", d.config.Namespace},
		{"configmaps", "get", d.config.Namespace},
		{"configmaps", "list", d.config.Namespace},

		// Cluster-scoped resources
		{"clusterroles", "create", ""},
		{"clusterrolebindings", "create", ""},

		// Cleanup permissions
		{"jobs", "delete", d.config.Namespace},
	}

	var missingPermissions []string

	for _, check := range requiredChecks {
		allowed, reason, err := d.checkPermission(ctx, check.resource, check.verb, check.namespace)
		if err != nil {
			return checks, fmt.Errorf("failed to check permission for %s %s: %w", check.verb, check.resource, err)
		}

		result := PermissionCheck{
			Resource:  check.resource,
			Verb:      check.verb,
			Namespace: check.namespace,
			Allowed:   allowed,
			Reason:    reason,
		}
		checks = append(checks, result)

		if !allowed {
			scope := "cluster-scoped"
			if check.namespace != "" {
				scope = fmt.Sprintf("namespace %q", check.namespace)
			}
			missingPermissions = append(missingPermissions,
				fmt.Sprintf("%s %s (%s)", check.verb, check.resource, scope))
		}
	}

	if len(missingPermissions) > 0 {
		return checks, fmt.Errorf("missing required permissions:\n  - %s",
			strings.Join(missingPermissions, "\n  - "))
	}

	return checks, nil
}

// checkPermission checks if the current user can perform the specified action.
func (d *Deployer) checkPermission(ctx context.Context, resource, verb, namespace string) (bool, string, error) {
	review := &authv1.SelfSubjectAccessReview{
		Spec: authv1.SelfSubjectAccessReviewSpec{
			ResourceAttributes: &authv1.ResourceAttributes{
				Verb:      verb,
				Resource:  resource,
				Namespace: namespace,
			},
		},
	}

	result, err := d.clientset.AuthorizationV1().SelfSubjectAccessReviews().Create(ctx, review, metav1.CreateOptions{})
	if err != nil {
		return false, "", err
	}

	return result.Status.Allowed, result.Status.Reason, nil
}
