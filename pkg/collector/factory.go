package collector

// Factory creates collectors with their dependencies.
// This interface enables dependency injection for testing.
type Factory interface {
	CreateKModCollector() Collector
	CreateSystemDCollector() Collector
	CreateGrubCollector() Collector
	CreateSysctlCollector() Collector
	CreateKubernetesCollector() Collector
	CreateImageCollector() Collector
	CreateSMICollector() Collector
}

// DefaultFactory creates collectors with production dependencies.
type DefaultFactory struct {
	SystemDServices []string
}

// NewDefaultFactory creates a factory with default settings.
func NewDefaultFactory() *DefaultFactory {
	return &DefaultFactory{
		SystemDServices: []string{
			"containerd.service",
			"docker.service",
			"kubelet.service",
		},
	}
}

// ComponentCollector creates a component collector.
func (f *DefaultFactory) CreateKModCollector() Collector {
	return &KModCollector{}
}

// CreateSMICollector creates an SMI collector.
func (f *DefaultFactory) CreateSMICollector() Collector {
	return &SMICollector{}
}

// CreateSystemDCollector creates a systemd collector.
func (f *DefaultFactory) CreateSystemDCollector() Collector {
	return &SystemDCollector{
		Services: f.SystemDServices,
	}
}

// CreateGrubCollector creates a GRUB collector.
func (f *DefaultFactory) CreateGrubCollector() Collector {
	return &GrubCollector{}
}

// CreateSysctlCollector creates a sysctl collector.
func (f *DefaultFactory) CreateSysctlCollector() Collector {
	return &SysctlCollector{}
}

// CreateKubernetesCollector creates a Kubernetes API collector.
func (f *DefaultFactory) CreateKubernetesCollector() Collector {
	return &KubernetesCollector{}
}

// CreateImageCollector creates a container image collector.
func (f *DefaultFactory) CreateImageCollector() Collector {
	return &ImageCollector{}
}
