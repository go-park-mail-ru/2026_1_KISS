package domain

type ContainerResourceStats struct {
	CPUPercent     float64
	MemoryUsage    int64
	MemoryLimit    int64
	MemoryPercent  float64
	CPUCores       uint32
	DiskLimitBytes int64
	GPUAvailable   bool
}
