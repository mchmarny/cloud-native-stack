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

package header

import (
	"time"
)

// Kind represents the type of CNS resource.
// All CNS resources should use these constants for consistency.
type Kind string

// Valid Kind constants for all CNS resource types.
const (
	KindSnapshot         Kind = "Snapshot"
	KindRecipe           Kind = "Recipe"
	KindRecipeResult     Kind = "RecipeResult"
	KindValidationResult Kind = "ValidationResult"
)

// String returns the string representation of the Kind.
func (k Kind) String() string {
	return string(k)
}

// IsValid checks if the Kind is one of the recognized kinds.
func (k *Kind) IsValid() bool {
	switch *k {
	case KindSnapshot, KindRecipe, KindRecipeResult, KindValidationResult:
		return true
	default:
		return false
	}
}

// Option is a functional option for configuring Header instances.
type Option func(*Header)

// WithMetadata returns an Option that adds a metadata key-value pair to the Header.
// If the Metadata map is nil, it will be initialized.
func WithMetadata(key, value string) Option {
	return func(h *Header) {
		if h.Metadata == nil {
			h.Metadata = make(map[string]string)
		}
		h.Metadata[key] = value
	}
}

// WithKind returns an Option that sets the Kind field of the Header.
// Kind represents the type of the resource (e.g., "Snapshot", "Recipe").
func WithKind(kind Kind) Option {
	return func(h *Header) {
		h.Kind = kind
	}
}

// WithAPIVersion returns an Option that sets the APIVersion field of the Header.
// The APIVersion identifies the schema version for the resource.
func WithAPIVersion(version string) Option {
	return func(h *Header) {
		h.APIVersion = version
	}
}

// SetKind updates the Kind field of the Header.
func (h *Header) SetKind(kind Kind) {
	h.Kind = kind
}

// GetKind returns the Kind field of the Header.
func (h *Header) GetKind() Kind {
	return h.Kind
}

// GetMetadata returns the Metadata map of the Header.
func (h *Header) GetMetadata() map[string]string {
	return h.Metadata
}

// New creates a new Header instance with the provided functional options.
// The Metadata map is initialized automatically.
func New(opts ...Option) *Header {
	s := &Header{
		Metadata: make(map[string]string),
	}

	// Apply options
	for _, opt := range opts {
		opt(s)
	}

	return s
}

// Header contains metadata and versioning information for CNS resources.
// It follows Kubernetes-style resource conventions with Kind, APIVersion, and Metadata fields.
type Header struct {
	// Kind is the type of the snapshot object.
	Kind Kind `json:"kind,omitempty" yaml:"kind,omitempty"`

	// APIVersion is the API version of the snapshot object.
	APIVersion string `json:"apiVersion,omitempty" yaml:"apiVersion,omitempty"`

	// Metadata contains key-value pairs with metadata about the snapshot.
	Metadata map[string]string `json:"metadata,omitempty" yaml:"metadata,omitempty"`
}

// Init initializes the Header with the specified kind, apiVersion, and version.
// It sets the Kind, APIVersion, and populates Metadata with timestamp and version.
// Uses unprefixed keys (timestamp, version) for all kinds.
func (h *Header) Init(kind Kind, apiVersion string, version string) {
	h.Kind = kind
	h.APIVersion = apiVersion
	h.Metadata = make(map[string]string)

	// Use unprefixed keys for all kinds
	h.Metadata["timestamp"] = time.Now().UTC().Format(time.RFC3339)
	if version != "" {
		h.Metadata["version"] = version
	}
}
