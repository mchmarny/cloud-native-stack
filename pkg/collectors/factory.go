package collectors

// CollectorFactory creates collectors with their dependencies.
// This interface enables dependency injection for testing.
type CollectorFactory interface {
	CreateKModCollector() Collector
	CreateSystemDCollector() Collector
	CreateGrubCollector() Collector
	CreateSysctlCollector() Collector
	CreateKubernetesCollector() Collector
	CreateImageCollector() Collector
	CreateSMICollector() Collector
}

// DefaultCollectorFactory creates collectors with production dependencies.
type DefaultCollectorFactory struct {
	SystemDServices []string
}

// NewDefaultCollectorFactory creates a factory with default settings.
func NewDefaultCollectorFactory() *DefaultCollectorFactory {
	return &DefaultCollectorFactory{
		SystemDServices: []string{
			"containerd.service",
		},
	}
}

// ComponentCollector creates a component collector.
func (f *DefaultCollectorFactory) CreateKModCollector() Collector {
	return &KModCollector{}
}

// CreateSMICollector creates an SMI collector.
func (f *DefaultCollectorFactory) CreateSMICollector() Collector {
	return &SMICollector{}
}

// CreateSystemDCollector creates a systemd collector.
func (f *DefaultCollectorFactory) CreateSystemDCollector() Collector {
	return &SystemDCollector{
		Services: f.SystemDServices,
	}
}

// CreateGrubCollector creates a GRUB collector.
func (f *DefaultCollectorFactory) CreateGrubCollector() Collector {
	return &GrubCollector{}
}

// CreateSysctlCollector creates a sysctl collector.
func (f *DefaultCollectorFactory) CreateSysctlCollector() Collector {
	return &SysctlCollector{}
}

// CreateKubernetesCollector creates a Kubernetes API collector.
func (f *DefaultCollectorFactory) CreateKubernetesCollector() Collector {
	return &KubernetesCollector{}
}

// CreateImageCollector creates a container image collector.
func (f *DefaultCollectorFactory) CreateImageCollector() Collector {
	return &ImageCollector{}
}
