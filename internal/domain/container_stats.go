package domain

import "time"

type ContainerResourceStats struct {
	CPUPercent     float64
	MemoryUsage    int64
	MemoryLimit    int64
	MemoryPercent  float64
	CPUCores       uint32
	DiskLimitBytes int64
	GPUAvailable   bool
}

type SessionStats struct {
	ContainerResourceStats
	QueuePosition     int32
	SnapshotAge       time.Duration
	SnapshotSizeBytes int64
	SessionState      string // "active" | "queued" | "inactive"
}
