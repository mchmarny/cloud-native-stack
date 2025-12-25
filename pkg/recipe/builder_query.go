package recipe

import "github.com/NVIDIA/cloud-native-stack/pkg/version"

// QueryBuilder provides a fluent interface for constructing Query objects.
type QueryBuilder struct {
	query Query
}

// NewQueryBuilder creates a new QueryBuilder with default values.
func NewQueryBuilder() *QueryBuilder {
	return &QueryBuilder{
		query: Query{
			Os:      OSAny,
			Service: ServiceAny,
			GPU:     GPUAny,
			Intent:  IntentAny,
		},
	}
}

// WithOS sets the operating system family.
func (b *QueryBuilder) WithOS(os OsFamily) *QueryBuilder {
	b.query.Os = os
	return b
}

// WithOSVersion sets the operating system family and version.
func (b *QueryBuilder) WithOSVersion(os OsFamily, osVersion string) *QueryBuilder {
	b.query.Os = os
	if osVersion != "" {
		if v, err := version.ParseVersion(osVersion); err == nil {
			b.query.OsVersion = &v
		}
	}
	return b
}

// WithKernel sets the kernel version.
func (b *QueryBuilder) WithKernel(kernel string) *QueryBuilder {
	if kernel != "" {
		if v, err := version.ParseVersion(kernel); err == nil {
			b.query.Kernel = &v
		}
	}
	return b
}

// WithService sets the managed service context.
func (b *QueryBuilder) WithService(service ServiceType) *QueryBuilder {
	b.query.Service = service
	return b
}

// WithK8s sets the Kubernetes version.
func (b *QueryBuilder) WithK8s(k8sVersion string) *QueryBuilder {
	if k8sVersion != "" {
		if v, err := version.ParseVersion(k8sVersion); err == nil {
			b.query.K8s = &v
		}
	}
	return b
}

// WithGPU sets the GPU type.
func (b *QueryBuilder) WithGPU(gpu GPUType) *QueryBuilder {
	b.query.GPU = gpu
	return b
}

// WithIntent sets the workload intent.
func (b *QueryBuilder) WithIntent(intent IntentType) *QueryBuilder {
	b.query.Intent = intent
	return b
}

// WithContext sets whether to include context metadata in responses.
func (b *QueryBuilder) WithContext(includeContext bool) *QueryBuilder {
	b.query.IncludeContext = includeContext
	return b
}

// Build returns the constructed Query.
func (b *QueryBuilder) Build() *Query {
	query := b.query
	return &query
}

// BuildAndValidate returns the constructed Query after validating it.
func (b *QueryBuilder) BuildAndValidate() (*Query, error) {
	query := b.Build()
	if err := query.Validate(); err != nil {
		return nil, err
	}
	return query, nil
}
