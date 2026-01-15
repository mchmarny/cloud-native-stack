/*
Copyright © 2025 NVIDIA Corporation
SPDX-License-Identifier: Apache-2.0
*/

package validator

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/NVIDIA/cloud-native-stack/pkg/recipe"
	"github.com/NVIDIA/cloud-native-stack/pkg/recipe/header"
	"github.com/NVIDIA/cloud-native-stack/pkg/snapshotter"
)

const (
	// APIVersion is the API version for validation results.
	APIVersion = "cns.nvidia.com/v1alpha1"
)

// Validator evaluates recipe constraints against snapshot measurements.
type Validator struct {
	// Version is the validator version (typically the CLI version).
	Version string
}

// Option is a functional option for configuring Validator instances.
type Option func(*Validator)

// WithVersion returns an Option that sets the Validator version string.
func WithVersion(version string) Option {
	return func(v *Validator) {
		v.Version = version
	}
}

// New creates a new Validator with the provided options.
func New(opts ...Option) *Validator {
	v := &Validator{}
	for _, opt := range opts {
		opt(v)
	}
	return v
}

// Validate evaluates all constraints from the recipe against the snapshot.
// Returns a ValidationResult containing per-constraint results and summary.
func (v *Validator) Validate(ctx context.Context, recipeResult *recipe.RecipeResult, snap *snapshotter.Snapshot) (*ValidationResult, error) {
	start := time.Now()

	if recipeResult == nil {
		return nil, fmt.Errorf("recipe cannot be nil")
	}
	if snap == nil {
		return nil, fmt.Errorf("snapshot cannot be nil")
	}

	result := NewValidationResult()
	result.Init(header.KindValidationResult, APIVersion, v.Version)

	// Evaluate each constraint
	for _, constraint := range recipeResult.Constraints {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		cv := v.evaluateConstraint(constraint, snap)
		result.Results = append(result.Results, cv)

		// Update summary counts
		switch cv.Status {
		case ConstraintStatusPassed:
			result.Summary.Passed++
		case ConstraintStatusFailed:
			result.Summary.Failed++
		case ConstraintStatusSkipped:
			result.Summary.Skipped++
		}
	}

	// Calculate summary
	result.Summary.Total = len(recipeResult.Constraints)
	result.Summary.Duration = time.Since(start)

	// Determine overall status
	switch {
	case result.Summary.Failed > 0:
		result.Summary.Status = ValidationStatusFail
	case result.Summary.Skipped > 0:
		result.Summary.Status = ValidationStatusPartial
	default:
		result.Summary.Status = ValidationStatusPass
	}

	slog.Debug("validation completed",
		"passed", result.Summary.Passed,
		"failed", result.Summary.Failed,
		"skipped", result.Summary.Skipped,
		"status", result.Summary.Status,
		"duration", result.Summary.Duration)

	return result, nil
}

// evaluateConstraint evaluates a single constraint against the snapshot.
func (v *Validator) evaluateConstraint(constraint recipe.Constraint, snap *snapshotter.Snapshot) ConstraintValidation {
	cv := ConstraintValidation{
		Name:     constraint.Name,
		Expected: constraint.Value,
	}

	// Parse the constraint path
	path, err := ParseConstraintPath(constraint.Name)
	if err != nil {
		cv.Status = ConstraintStatusSkipped
		cv.Message = fmt.Sprintf("invalid constraint path: %v", err)
		slog.Warn("skipping constraint with invalid path",
			"name", constraint.Name,
			"error", err)
		return cv
	}

	// Extract the actual value from snapshot
	actual, err := path.ExtractValue(snap)
	if err != nil {
		cv.Status = ConstraintStatusSkipped
		cv.Message = fmt.Sprintf("value not found in snapshot: %v", err)
		slog.Warn("skipping constraint - value not found",
			"name", constraint.Name,
			"path", path.String(),
			"error", err)
		return cv
	}
	cv.Actual = actual

	// Print detected criteria based on the path and value found
	printDetectedCriteria(path.String(), actual)

	// Parse the constraint expression
	parsed, err := ParseConstraintExpression(constraint.Value)
	if err != nil {
		cv.Status = ConstraintStatusSkipped
		cv.Message = fmt.Sprintf("invalid constraint expression: %v", err)
		slog.Warn("skipping constraint with invalid expression",
			"name", constraint.Name,
			"expression", constraint.Value,
			"error", err)
		return cv
	}

	// Evaluate the constraint
	passed, err := parsed.Evaluate(actual)
	if err != nil {
		cv.Status = ConstraintStatusFailed
		cv.Message = fmt.Sprintf("evaluation failed: %v", err)
		slog.Debug("constraint evaluation failed",
			"name", constraint.Name,
			"expected", constraint.Value,
			"actual", actual,
			"error", err)
		return cv
	}

	if passed {
		cv.Status = ConstraintStatusPassed
		slog.Debug("constraint passed",
			"name", constraint.Name,
			"expected", constraint.Value,
			"actual", actual)
	} else {
		cv.Status = ConstraintStatusFailed
		cv.Message = fmt.Sprintf("expected %s, got %s", constraint.Value, actual)
		slog.Debug("constraint failed",
			"name", constraint.Name,
			"expected", constraint.Value,
			"actual", actual)
	}

	return cv
}

// printDetectedCriteria prints detected criteria based on the constraint path and value.
func printDetectedCriteria(path, value string) {
	switch path {
	case "K8s.server.version":
		slog.Info("detected criteria", "service", value)
	case "GPU.smi.gpu.model":
		slog.Info("detected criteria", "accelerator", value)
	case "OS.release.ID":
		slog.Info("detected criteria", "os", value)
	case "OS.release.VERSION_ID":
		slog.Info("detected criteria", "os_version", value)
	}
}
