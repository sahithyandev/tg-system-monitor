package metrics

import "time"

type MetricSample struct {
	Timestamp   time.Time
	CPUPercent  float64
	MemPercent  float64
	SwapPercent float64
	DiskPercent float64
	Load1       float64
	Load5       float64
	Load15      float64
	Uptime      float64
}

type Process struct {
	PID    int
	Name   string
	CPU    float64
	Memory uint64 // RSS in KB
}
