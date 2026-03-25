package formatter

import (
	"strings"
	"testing"
	"tg-system-monitor/metrics"
	"time"
)

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		kb       uint64
		expected string
	}{
		{0, "0 B"},
		{1, "1.0 KB"},
		{1024, "1.0 MB"},
		{1024 * 1024, "1.0 GB"},
		{500, "500.0 KB"},
		{1500, "1.5 MB"},
	}

	for _, tt := range tests {
		result := FormatBytes(tt.kb)
		if result != tt.expected {
			t.Errorf("FormatBytes(%d) = %s; want %s", tt.kb, result, tt.expected)
		}
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		seconds  float64
		expected string
	}{
		{30, "00m 30s"},
		{90, "01m 30s"},
		{3600, "01h 00m 00s"},
		{3661, "01h 01m 01s"},
		{86400, "1d 00h 00m"},
		{90061, "1d 01h 01m"},
	}

	for _, tt := range tests {
		result := FormatDuration(tt.seconds)
		if result != tt.expected {
			t.Errorf("FormatDuration(%f) = %s; want %s", tt.seconds, result, tt.expected)
		}
	}
}

func TestFormatStatus(t *testing.T) {
	sample := &metrics.MetricSample{
		Timestamp:   time.Date(2026, 3, 25, 14, 10, 3, 0, time.UTC),
		Uptime:      3661,
		CPUPercent:  22.4,
		MemPercent:  61.8,
		SwapPercent: 0.0,
		DiskPercent: 74.1,
		Load1:       0.84,
		Load5:       0.92,
		Load15:      0.88,
	}

	result := FormatStatus("web-01", sample)

	expectedParts := []string{
		"Host: web-01",
		"Uptime: 01h 01m 01s",
		"CPU: 22.4%",
		"Memory: 61.8%",
		"Load avg: 0.84 0.92 0.88",
		"Updated: 2026-03-25 14:10:03",
	}

	for _, part := range expectedParts {
		if !strings.Contains(result, part) {
			t.Errorf("FormatStatus output missing expected part: %s\nOutput:\n%s", part, result)
		}
	}
}

func TestFormatTop(t *testing.T) {
	procs := []metrics.Process{
		{PID: 1234, Name: "java", Memory: 713 * 1024},
		{PID: 2441, Name: "nginx", Memory: 18 * 1024},
	}

	result := FormatTop(procs)

	expectedParts := []string{
		"Top processes by memory:",
		"1. java (1234): 713.0 MB",
		"2. nginx (2441): 18.0 MB",
	}

	for _, part := range expectedParts {
		if !strings.Contains(result, part) {
			t.Errorf("FormatTop output missing expected part: %s\nOutput:\n%s", part, result)
		}
	}
}
