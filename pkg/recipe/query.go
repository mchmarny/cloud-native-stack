// Package recipe implements a rule-based configuration system for generating
// optimized system configurations based on environment context.
//
// Query Matching:
//
// The recipe system uses an asymmetric matching algorithm where rules (overlays)
// match against candidate queries. A rule matches a candidate when every populated
// field in the rule is satisfied by the candidate:
//
//   - Empty/zero fields in the rule act as wildcards (match anything)
//   - Matching is asymmetric: rule.IsMatch(candidate) ≠ candidate.IsMatch(rule)
//
// Example:
//
//	rule := Query{Os: OSUbuntu}  // wildcard for other fields
//	candidate := Query{Os: OSUbuntu, GPU: GPUH100}
//	rule.IsMatch(candidate)      // true - rule matches broader candidate
//	candidate.IsMatch(rule)      // false - candidate too specific for rule
//
// Recipe Construction:
//
// Recipes are built by:
//  1. Starting with base measurements (apply to all queries)
//  2. Layering matching overlays on top (environment-specific)
//  3. Deep cloning all data to prevent mutation
//  4. Optionally stripping context metadata if not requested
package recipe

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/NVIDIA/cloud-native-stack/pkg/recipe/version"
)

const (
	anyValue = "any"
)

// Query represents a recommendation query with various context parameters.
type Query struct {
	// Os is the operating system family (e.g., ubuntu, cos)
	Os OsFamily `json:"os,omitempty" yaml:"os,omitempty"`

	// OsVersion is the operating system version (e.g., 22.04)
	OsVersion *version.Version `json:"osv,omitempty" yaml:"osv,omitempty"`

	// Kernel is the running kernel version (e.g., 5.15.0)
	Kernel *version.Version `json:"kernel,omitempty" yaml:"kernel,omitempty"`

	// Service is the managed service context (e.g., eks, gke, or self-managed)
	Service ServiceType `json:"service,omitempty" yaml:"service,omitempty"`

	// K8s is the Kubernetes cluster version (e.g., v1.25.4)
	K8s *version.Version `json:"k8s,omitempty" yaml:"k8s,omitempty"`

	// GPU is the GPU type (e.g., H100, GB200)
	GPU GPUType `json:"gpu,omitempty" yaml:"gpu,omitempty"`

	// Intent is the workload intent (e.g., training or inference)
	Intent IntentType `json:"intent,omitempty" yaml:"intent,omitempty"`

	// IncludeContext indicates whether to include context metadata in the response
	IncludeContext bool `json:"withContext,omitempty" yaml:"withContext,omitempty"`
}

func (q *Query) IsEmpty() bool {
	return (q.Os == "" || q.Os == anyValue) &&
		(q.OsVersion == nil || !q.OsVersion.IsValid()) &&
		(q.Kernel == nil || !q.Kernel.IsValid()) &&
		(q.Service == "" || q.Service == anyValue) &&
		(q.K8s == nil || !q.K8s.IsValid()) &&
		(q.GPU == "" || q.GPU == anyValue) &&
		(q.Intent == "" || q.Intent == anyValue)
}

// MarshalJSON implements custom JSON marshaling for Query.
// It omits fields that are set to their default "any" value to produce cleaner JSON output.
func (q Query) MarshalJSON() ([]byte, error) {
	aux := struct {
		Os             *OsFamily    `json:"os,omitempty"`
		OsVersion      *string      `json:"osv,omitempty"`
		Kernel         *string      `json:"kernel,omitempty"`
		Service        *ServiceType `json:"service,omitempty"`
		K8s            *string      `json:"k8s,omitempty"`
		GPU            *GPUType     `json:"gpu,omitempty"`
		Intent         *IntentType  `json:"intent,omitempty"`
		IncludeContext *bool        `json:"withContext,omitempty"`
	}{}

	// Only include non-empty and non-"any" enum values
	if q.Os != "" && q.Os != OSAny {
		aux.Os = &q.Os
	}
	if q.OsVersion != nil {
		v := q.OsVersion.String()
		aux.OsVersion = &v
	}
	if q.Kernel != nil {
		v := q.Kernel.String()
		aux.Kernel = &v
	}
	if q.Service != "" && q.Service != ServiceAny {
		aux.Service = &q.Service
	}
	if q.K8s != nil {
		v := q.K8s.String()
		aux.K8s = &v
	}
	if q.GPU != "" && q.GPU != GPUAny {
		aux.GPU = &q.GPU
	}
	if q.Intent != "" && q.Intent != IntentAny {
		aux.Intent = &q.Intent
	}
	if q.IncludeContext {
		aux.IncludeContext = &q.IncludeContext
	}

	return json.Marshal(aux)
}

// UnmarshalJSON implements custom JSON unmarshaling for Query.
// It handles the omitted "any" values by treating missing fields as wildcards.
func (q *Query) UnmarshalJSON(data []byte) error {
	aux := struct {
		Os             *OsFamily    `json:"os"`
		OsVersion      *string      `json:"osv"`
		Kernel         *string      `json:"kernel"`
		Service        *ServiceType `json:"service"`
		K8s            *string      `json:"k8s"`
		GPU            *GPUType     `json:"gpu"`
		Intent         *IntentType  `json:"intent"`
		IncludeContext *bool        `json:"withContext"`
	}{}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	// Set fields from aux, using defaults for missing values
	if aux.Os != nil {
		q.Os = *aux.Os
	} else {
		q.Os = OSAny
	}

	if aux.OsVersion != nil {
		v, err := version.ParseVersion(*aux.OsVersion)
		if err != nil {
			return fmt.Errorf("invalid osv: %w", err)
		}
		q.OsVersion = &v
	} else {
		q.OsVersion = nil
	}

	if aux.Kernel != nil {
		v, err := version.ParseVersion(*aux.Kernel)
		if err != nil {
			return fmt.Errorf("invalid kernel: %w", err)
		}
		q.Kernel = &v
	} else {
		q.Kernel = nil
	}

	if aux.Service != nil {
		q.Service = *aux.Service
	} else {
		q.Service = ServiceAny
	}

	if aux.K8s != nil {
		v, err := version.ParseVersion(*aux.K8s)
		if err != nil {
			return fmt.Errorf("invalid k8s: %w", err)
		}
		q.K8s = &v
	} else {
		q.K8s = nil
	}

	if aux.GPU != nil {
		q.GPU = *aux.GPU
	} else {
		q.GPU = GPUAny
	}

	if aux.Intent != nil {
		q.Intent = *aux.Intent
	} else {
		q.Intent = IntentAny
	}

	if aux.IncludeContext != nil {
		q.IncludeContext = *aux.IncludeContext
	}

	return nil
}

// Validate checks if the query has valid field values and combinations.
func (q *Query) Validate() error {
	if q == nil {
		return fmt.Errorf("query cannot be nil")
	}

	// Validate enum types
	if q.Os != "" && q.Os != OSAny && !q.Os.IsValid() {
		return fmt.Errorf("invalid os family: %s", q.Os)
	}
	if q.Service != "" && q.Service != ServiceAny && !q.Service.IsValid() {
		return fmt.Errorf("invalid service type: %s", q.Service)
	}
	if q.GPU != "" && q.GPU != GPUAny && !q.GPU.IsValid() {
		return fmt.Errorf("invalid gpu type: %s", q.GPU)
	}
	if q.Intent != "" && q.Intent != IntentAny && !q.Intent.IsValid() {
		return fmt.Errorf("invalid intent type: %s", q.Intent)
	}

	return nil
}

// IsMatch reports whether the rule (receiver) applies to the candidate query, only
// matching when every populated rule field is satisfied by the candidate.
func (q *Query) IsMatch(other *Query) bool {
	if q == nil || other == nil || other.IsEmpty() {
		return false
	}
	if !matchEnum(q.Os, other.Os, OSAny) {
		return false
	}
	if !matchVersion(q.OsVersion, other.OsVersion) {
		return false
	}
	if !matchVersion(q.Kernel, other.Kernel) {
		return false
	}
	if !matchEnum(q.Service, other.Service, ServiceAny) {
		return false
	}
	if !matchVersion(q.K8s, other.K8s) {
		return false
	}
	if !matchEnum(q.GPU, other.GPU, GPUAny) {
		return false
	}
	if !matchEnum(q.Intent, other.Intent, IntentAny) {
		return false
	}
	return true
}

// String returns a human-readable representation of the query.
func (q *Query) String() string {
	return fmt.Sprintf("OS: %s %s, Kernel: %s, Service: %s, K8s: %s, GPU: %s, Intent: %s, Context: %t",
		normalizeValue(q.Os),
		normalizeVersionValue(q.OsVersion),
		normalizeVersionValue(q.Kernel),
		normalizeValue(q.Service),
		normalizeVersionValue(q.K8s),
		normalizeValue(q.GPU),
		normalizeValue(q.Intent),
		q.IncludeContext,
	)
}

// normalizeValue normalizes a string value for key generation.
// If the value is empty or only whitespace, it returns "any".
func normalizeValue[T ~string](val T) string {
	var zero T
	if val == zero {
		return anyValue
	}
	v := strings.TrimSpace(string(val))
	if v == "" {
		return anyValue
	}
	return strings.ToLower(v)
}

// normalizeVersionValue normalizes a version value for key generation.
// If the version is nil or invalid (zero/unset), it returns "any".
func normalizeVersionValue(val *version.Version) string {
	if val == nil || !val.IsValid() {
		return anyValue
	}
	return normalizeValue(strings.TrimSpace(val.String()))
}

// QueryParameterType represents the type of query parameter.
const (
	QueryParamOSFamily       string = "os"
	QueryParamOSVersion      string = "osv"
	QueryParamKernel         string = "kernel"
	QueryParamService        string = "service"
	QueryParamEnvironment    string = "env"
	QueryParamKubernetes     string = "k8s"
	QueryParamGPU            string = "gpu"
	QueryParamIntent         string = "intent"
	QueryParamIncludeContext string = "context"
)

// OsFamily represents the operating system family.
type OsFamily string

const (
	OSAny    OsFamily = anyValue
	OSUbuntu OsFamily = "ubuntu"
	OSCOS    OsFamily = "cos"
)

// String returns the string representation of the OS family.
func (o OsFamily) String() string {
	return string(o)
}

// IsValid returns true if the OS family is a valid supported value.
func (o OsFamily) IsValid() bool {
	switch o {
	case OSAny, OSUbuntu, OSCOS:
		return true
	default:
		return false
	}
}

// SupportedOSFamilies returns all supported OS family values.
func SupportedOSFamilies() []OsFamily {
	return []OsFamily{OSAny, OSUbuntu, OSCOS}
}

// ParseOsFamily parses the OS family from query parameters.
func ParseOsFamily(list url.Values) (OsFamily, error) {
	osStr := strings.ToLower(list.Get(QueryParamOSFamily))
	if osStr == "" {
		return OSAny, nil
	}

	os := OsFamily(osStr)
	if !os.IsValid() {
		supported := make([]string, 0, len(SupportedOSFamilies()))
		for _, s := range SupportedOSFamilies() {
			supported = append(supported, s.String())
		}
		return "", fmt.Errorf("invalid os family: %s, supported: %s", osStr, strings.Join(supported, ", "))
	}
	return os, nil
}

// ServiceType represents the managed service context.
type ServiceType string

const (
	ServiceAny ServiceType = anyValue
	ServiceEKS ServiceType = "eks"
	ServiceGKE ServiceType = "gke"
	ServiceAKS ServiceType = "aks"
)

// String returns the string representation of the service type.
func (s ServiceType) String() string {
	return string(s)
}

// IsValid returns true if the service type is a valid supported value.
func (s ServiceType) IsValid() bool {
	switch s {
	case ServiceAny, ServiceEKS, ServiceGKE, ServiceAKS:
		return true
	default:
		return false
	}
}

// SupportedServiceTypes returns all supported service type values.
func SupportedServiceTypes() []ServiceType {
	return []ServiceType{ServiceAny, ServiceEKS, ServiceGKE, ServiceAKS}
}

// ParseServiceType parses the service type from query parameters.
func ParseServiceType(list url.Values) (ServiceType, error) {
	svcStr := strings.ToLower(list.Get(QueryParamService))
	if svcStr == "" {
		svcStr = strings.ToLower(list.Get(QueryParamEnvironment))
	}
	if svcStr == "" {
		return ServiceAny, nil
	}

	svc := ServiceType(svcStr)
	if !svc.IsValid() {
		supported := make([]string, 0, len(SupportedServiceTypes()))
		for _, s := range SupportedServiceTypes() {
			supported = append(supported, s.String())
		}
		return "", fmt.Errorf("invalid service type: %s, supported: %s", svcStr, strings.Join(supported, ", "))
	}
	return svc, nil
}

// GPUType represents the GPU type.
type GPUType string

const (
	GPUAny  GPUType = anyValue
	GPUH100 GPUType = "h100"
	GPUB200 GPUType = "gb200"
)

// String returns the string representation of the GPU type.
func (g GPUType) String() string {
	return string(g)
}

// IsValid returns true if the GPU type is a valid supported value.
func (g GPUType) IsValid() bool {
	switch g {
	case GPUAny, GPUH100, GPUB200:
		return true
	default:
		return false
	}
}

// SupportedGPUTypes returns all supported GPU type values.
func SupportedGPUTypes() []GPUType {
	return []GPUType{GPUAny, GPUH100, GPUB200}
}

// ParseGPUType parses the GPU type from query parameters.
func ParseGPUType(list url.Values) (GPUType, error) {
	gpuStr := strings.ToLower(list.Get(QueryParamGPU))
	if gpuStr == "" {
		return GPUAny, nil
	}

	gpu := GPUType(gpuStr)
	if !gpu.IsValid() {
		supported := make([]string, 0, len(SupportedGPUTypes()))
		for _, g := range SupportedGPUTypes() {
			supported = append(supported, g.String())
		}
		return "", fmt.Errorf("invalid gpu type: %s, supported: %s", gpuStr, strings.Join(supported, ", "))
	}
	return gpu, nil
}

// IntentType represents the workload intent.
type IntentType string

const (
	IntentAny       IntentType = anyValue
	IntentTraining  IntentType = "training"
	IntentInference IntentType = "inference"
)

// String returns the string representation of the intent type.
func (i IntentType) String() string {
	return string(i)
}

// IsValid returns true if the intent type is a valid supported value.
func (i IntentType) IsValid() bool {
	switch i {
	case IntentAny, IntentTraining, IntentInference:
		return true
	default:
		return false
	}
}

// SupportedIntentTypes returns all supported intent type values.
func SupportedIntentTypes() []IntentType {
	return []IntentType{IntentAny, IntentTraining, IntentInference}
}

// ParseIntentType parses the intent type from query parameters.
func ParseIntentType(list url.Values) (IntentType, error) {
	intentStr := strings.ToLower(list.Get(QueryParamIntent))
	if intentStr == "" {
		return IntentAny, nil
	}

	intent := IntentType(intentStr)
	if !intent.IsValid() {
		supported := make([]string, 0, len(SupportedIntentTypes()))
		for _, i := range SupportedIntentTypes() {
			supported = append(supported, i.String())
		}
		return "", fmt.Errorf("invalid intent type: %s, supported: %s", intentStr, strings.Join(supported, ", "))
	}
	return intent, nil
}

// ParseQuery parses an HTTP request into a Query struct.
func ParseQuery(r *http.Request) (*Query, error) {
	u := r.URL.Query()
	q := &Query{}

	var err error

	// Parse OS family
	if q.Os, err = ParseOsFamily(u); err != nil {
		return nil, err
	}

	// Parse OS version
	if osVerStr := u.Get(QueryParamOSVersion); osVerStr != "" {
		var osVer version.Version
		if osVer, err = version.ParseVersion(osVerStr); err != nil {
			if errors.Is(err, version.ErrNegativeComponent) {
				return nil, fmt.Errorf("os version cannot contain negative numbers: %s", osVerStr)
			}
			return nil, fmt.Errorf("invalid os version %q: %w", osVerStr, err)
		}
		q.OsVersion = &osVer
	}

	// Parse kernel version
	if kernelStr := u.Get(QueryParamKernel); kernelStr != "" {
		var kernel version.Version
		if kernel, err = version.ParseVersion(kernelStr); err != nil {
			if errors.Is(err, version.ErrNegativeComponent) {
				return nil, fmt.Errorf("kernel version cannot contain negative numbers: %s", kernelStr)
			}
			return nil, fmt.Errorf("invalid kernel version %q: %w", kernelStr, err)
		}
		q.Kernel = &kernel
	}

	// Parse service type
	if q.Service, err = ParseServiceType(u); err != nil {
		return nil, err
	}

	// Parse Kubernetes version
	if k8sStr := u.Get(QueryParamKubernetes); k8sStr != "" {
		var k8sVer version.Version
		if k8sVer, err = version.ParseVersion(k8sStr); err != nil {
			if errors.Is(err, version.ErrNegativeComponent) {
				return nil, fmt.Errorf("kubernetes version cannot contain negative numbers: %s", k8sStr)
			}
			return nil, fmt.Errorf("invalid kubernetes version %q: %w", k8sStr, err)
		}
		q.K8s = &k8sVer
	}

	// Parse GPU type
	if q.GPU, err = ParseGPUType(u); err != nil {
		return nil, err
	}

	// Parse intent type
	if q.Intent, err = ParseIntentType(u); err != nil {
		return nil, err
	}

	// Parse context inclusion flag
	if contextStr := u.Get(QueryParamIncludeContext); contextStr != "" {
		q.IncludeContext = contextStr == "true" || contextStr == "1"
	}

	return q, nil
}

func matchEnum[T ~string](rule, candidate, wildcard T) bool {
	var zero T
	if rule == zero || rule == wildcard {
		return true
	}
	if candidate == zero || candidate == wildcard {
		return false
	}
	return rule == candidate
}

func matchVersion(rule, candidate *version.Version) bool {
	if rule == nil || !rule.IsValid() {
		return true
	}
	if candidate == nil || !candidate.IsValid() {
		return false
	}
	return *rule == *candidate
}
