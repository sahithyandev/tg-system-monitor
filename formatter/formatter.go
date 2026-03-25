package formatter

import (
	"fmt"
	"strings"
	"tg-system-monitor/metrics"
	"time"
)

func FormatBytes(kb uint64) string {
	bytes := kb * 1024
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := uint64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

func FormatDuration(seconds float64) string {
	d := time.Duration(seconds) * time.Second
	days := int(d.Hours()) / 24
	hours := int(d.Hours()) % 24
	minutes := int(d.Minutes()) % 60
	seconds_remain := int(d.Seconds()) % 60

	if days > 0 {
		return fmt.Sprintf("%dd %02dh %02dm", days, hours, minutes)
	}
	if hours > 0 {
		return fmt.Sprintf("%02dh %02dm %02ds", hours, minutes, seconds_remain)
	}
	return fmt.Sprintf("%02dm %02ds", minutes, seconds_remain)
}

func FormatStatus(hostname string, sample *metrics.MetricSample) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Host: %s\n", hostname))
	sb.WriteString(fmt.Sprintf("Uptime: %s\n\n", FormatDuration(sample.Uptime)))

	sb.WriteString(fmt.Sprintf("CPU: %.1f%%\n", sample.CPUPercent))
	sb.WriteString(fmt.Sprintf("Memory: %.1f%%\n", sample.MemPercent))
	sb.WriteString(fmt.Sprintf("Swap: %.1f%%\n", sample.SwapPercent))
	sb.WriteString(fmt.Sprintf("Disk /: %.1f%%\n", sample.DiskPercent))
	sb.WriteString(fmt.Sprintf("Load avg: %.2f %.2f %.2f\n\n", sample.Load1, sample.Load5, sample.Load15))

	sb.WriteString("Active alerts: none\n") // Placeholder for now
	sb.WriteString(fmt.Sprintf("Updated: %s", sample.Timestamp.Format("2006-01-02 15:04:05")))

	return sb.String()
}

func FormatTop(procs []metrics.Process) string {
	var sb strings.Builder
	sb.WriteString("Top processes by memory:\n")
	for i, p := range procs {
		sb.WriteString(fmt.Sprintf("%d. %s (%d): %s\n", i+1, p.Name, p.PID, FormatBytes(p.Memory)))
	}
	return sb.String()
}
