package recommendation

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/NVIDIA/cloud-native-stack/pkg/version"
)

const (
	anyValue = "any"
)

// Query represents a recommendation query with various context parameters.
type Query struct {
	// Os is the operating system family (e.g., ubuntu, cos)
	Os OsFamily `json:"os,omitempty"`

	// OsVersion is the operating system version (e.g., 22.04)
	OsVersion version.Version `json:"osv,omitempty"`

	// Kernel is the running kernel version (e.g., 5.15.0)
	Kernel version.Version `json:"kernel,omitempty"`

	// Service is the managed service context (e.g., eks, gke, or self-managed)
	Service ServiceType `json:"service,omitempty"`

	// K8s is the Kubernetes cluster version (e.g., v1.25.4)
	K8s version.Version `json:"k8s,omitempty"`

	// GPU is the GPU type (e.g., H100, GB200)
	GPU GPUType `json:"gpu,omitempty"`

	// Intent is the workload intent (e.g., training or inference)
	Intent IntentType `json:"intent,omitempty"`
}

// IsMatch checks if the query matches another query, treating "any" values as wildcards.
func (q *Query) IsMatch(other *Query) bool {
	leftOS := q.Os
	if leftOS == "" {
		leftOS = OSAny
	}
	rightOS := other.Os
	if rightOS == "" {
		rightOS = OSAny
	}
	if leftOS != OSAny && rightOS != OSAny && leftOS != rightOS {
		return false
	}
	if q.OsVersion.Precision != 0 && other.OsVersion.Precision != 0 && q.OsVersion != other.OsVersion {
		return false
	}
	if q.Kernel.Precision != 0 && other.Kernel.Precision != 0 && q.Kernel != other.Kernel {
		return false
	}
	leftService := q.Service
	if leftService == "" {
		leftService = ServiceAny
	}
	rightService := other.Service
	if rightService == "" {
		rightService = ServiceAny
	}
	if leftService != ServiceAny && rightService != ServiceAny && leftService != rightService {
		return false
	}
	if q.K8s.Precision != 0 && other.K8s.Precision != 0 && q.K8s != other.K8s {
		return false
	}
	leftGPU := q.GPU
	if leftGPU == "" {
		leftGPU = GPUAny
	}
	rightGPU := other.GPU
	if rightGPU == "" {
		rightGPU = GPUAny
	}
	if leftGPU != GPUAny && rightGPU != GPUAny && leftGPU != rightGPU {
		return false
	}
	leftIntent := q.Intent
	if leftIntent == "" {
		leftIntent = IntentAny
	}
	rightIntent := other.Intent
	if rightIntent == "" {
		rightIntent = IntentAny
	}
	if leftIntent != IntentAny && rightIntent != IntentAny && leftIntent != rightIntent {
		return false
	}
	return true
}

// String returns a human-readable representation of the query.
func (q *Query) String() string {
	return fmt.Sprintf("OS: %s %s, Kernel: %s, Service: %s, K8s: %s, GPU: %s, Intent: %s",
		normalizeValue(q.Os),
		normalizeVersionValue(q.OsVersion),
		normalizeVersionValue(q.Kernel),
		normalizeValue(q.Service),
		normalizeVersionValue(q.K8s),
		normalizeValue(q.GPU),
		normalizeValue(q.Intent),
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
// If the version has zero precision, it returns "any".
func normalizeVersionValue(val version.Version) string {
	if val.Precision == 0 {
		return anyValue
	}
	return normalizeValue(strings.TrimSpace(val.String()))
}

// QueryParameterType represents the type of query parameter.
const (
	QueryParamOSFamily    string = "os"
	QueryParamOSVersion   string = "osv"
	QueryParamKernel      string = "kernel"
	QueryParamEnvironment string = "env"
	QueryParamKubernetes  string = "k8s"
	QueryParamGPU         string = "gpu"
	QueryParamIntent      string = "intent"
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
)

// String returns the string representation of the service type.
func (s ServiceType) String() string {
	return string(s)
}

// IsValid returns true if the service type is a valid supported value.
func (s ServiceType) IsValid() bool {
	switch s {
	case ServiceAny, ServiceEKS, ServiceGKE:
		return true
	default:
		return false
	}
}

// SupportedServiceTypes returns all supported service type values.
func SupportedServiceTypes() []ServiceType {
	return []ServiceType{ServiceAny, ServiceEKS, ServiceGKE}
}

// ParseServiceType parses the service type from query parameters.
func ParseServiceType(list url.Values) (ServiceType, error) {
	svcStr := strings.ToLower(list.Get(QueryParamEnvironment))
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
		if q.OsVersion, err = version.ParseVersion(osVerStr); err != nil {
			return nil, fmt.Errorf("invalid os version: %w", err)
		}
	}

	// Parse kernel version
	if kernelStr := u.Get(QueryParamKernel); kernelStr != "" {
		if q.Kernel, err = version.ParseVersion(kernelStr); err != nil {
			return nil, fmt.Errorf("invalid kernel version: %w", err)
		}
	}

	// Parse service type
	if q.Service, err = ParseServiceType(u); err != nil {
		return nil, err
	}

	// Parse Kubernetes version
	if k8sStr := u.Get(QueryParamKubernetes); k8sStr != "" {
		if q.K8s, err = version.ParseVersion(k8sStr); err != nil {
			return nil, fmt.Errorf("invalid kubernetes version: %w", err)
		}
	}

	// Parse GPU type
	if q.GPU, err = ParseGPUType(u); err != nil {
		return nil, err
	}

	// Parse intent type
	if q.Intent, err = ParseIntentType(u); err != nil {
		return nil, err
	}

	return q, nil
}
