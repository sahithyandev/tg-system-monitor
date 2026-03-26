//go:build darwin
// +build darwin

package metrics

import (
	"bytes"
	"fmt"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"
)

type Collector struct{}

func NewCollector() *Collector {
	return &Collector{}
}

func (c *Collector) Collect() (*MetricSample, error) {
	sample := &MetricSample{
		Timestamp: time.Now(),
	}

	var err error

	// CPU
	sample.CPUPercent, err = getCPUUsage()
	if err != nil {
		return nil, fmt.Errorf("failed to get CPU usage: %w", err)
	}

	// Memory & Swap
	sample.MemPercent, sample.SwapPercent, err = getMemoryUsage()
	if err != nil {
		return nil, fmt.Errorf("failed to get memory usage: %w", err)
	}

	// Load Average
	sample.Load1, sample.Load5, sample.Load15, err = getLoadAvg()
	if err != nil {
		return nil, fmt.Errorf("failed to get load avg: %w", err)
	}

	// Uptime
	sample.Uptime, err = getUptime()
	if err != nil {
		return nil, fmt.Errorf("failed to get uptime: %w", err)
	}

	// Disk
	sample.DiskPercent, err = getDiskUsage("/")
	if err != nil {
		return nil, fmt.Errorf("failed to get disk usage: %w", err)
	}

	return sample, nil
}

func getCPUUsage() (float64, error) {
	// Using top -l 1 -n 0 to get a quick snapshot of CPU usage
	cmd := exec.Command("top", "-l", "1", "-n", "0")
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		return 0, err
	}

	// Parse "CPU usage: 12.34% user, 5.67% sys, 81.99% idle"
	lines := strings.Split(out.String(), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "CPU usage:") {
			fields := strings.Fields(line)
			if len(fields) >= 7 {
				idleStr := strings.TrimSuffix(fields[6], "%")
				idle, err := strconv.ParseFloat(idleStr, 64)
				if err != nil {
					return 0, err
				}
				return 100 - idle, nil
			}
		}
	}

	return 0, fmt.Errorf("could not parse CPU usage from top")
}

func getMemoryUsage() (float64, float64, error) {
	// Memory usage via vm_stat
	cmd := exec.Command("vm_stat")
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		return 0, 0, err
	}

	var free, active, inactive, wired, compressed uint64
	lines := strings.Split(out.String(), "\n")
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		valStr := strings.TrimSuffix(fields[len(fields)-1], ".")
		val, _ := strconv.ParseUint(valStr, 10, 64)

		switch {
		case strings.HasPrefix(line, "Pages free:"):
			free = val
		case strings.HasPrefix(line, "Pages active:"):
			active = val
		case strings.HasPrefix(line, "Pages inactive:"):
			inactive = val
		case strings.HasPrefix(line, "Pages wired down:"):
			wired = val
		case strings.HasPrefix(line, "Pages occupied by compressor:"):
			compressed = val
		}
	}

	// Page size is typically 4096 on macOS
	pageSize := uint64(4096)
	total := (free + active + inactive + wired + compressed) * pageSize
	used := (active + wired + compressed) * pageSize

	memPercent := 0.0
	if total > 0 {
		memPercent = float64(used) / float64(total) * 100
	}

	// Swap usage via sysctl vm.swapusage
	swapOut, err := exec.Command("sysctl", "-n", "vm.swapusage").Output()
	if err == nil {
		// total = 2048.00M  used = 123.45M  free = 1924.55M  (encrypted)
		fields := strings.Fields(string(swapOut))
		if len(fields) >= 7 {
			totalSwap, _ := strconv.ParseFloat(strings.TrimSuffix(fields[2], "M"), 64)
			usedSwap, _ := strconv.ParseFloat(strings.TrimSuffix(fields[5], "M"), 64)
			if totalSwap > 0 {
				return memPercent, (usedSwap / totalSwap) * 100, nil
			}
		}
	}

	return memPercent, 0, nil
}

func getLoadAvg() (float64, float64, float64, error) {
	out, err := exec.Command("sysctl", "-n", "vm.loadavg").Output()
	if err != nil {
		return 0, 0, 0, err
	}

	// { 2.45 2.12 2.01 }
	fields := strings.Fields(string(out))
	if len(fields) < 4 {
		return 0, 0, 0, fmt.Errorf("invalid loadavg format")
	}

	l1, _ := strconv.ParseFloat(fields[1], 64)
	l5, _ := strconv.ParseFloat(fields[2], 64)
	l15, _ := strconv.ParseFloat(fields[3], 64)

	return l1, l5, l15, nil
}

func getUptime() (float64, error) {
	out, err := exec.Command("sysctl", "-n", "kern.boottime").Output()
	if err != nil {
		return 0, err
	}

	// { sec = 1711364400, usec = 0 } Mon Mar 25 ...
	parts := strings.Split(string(out), ",")
	if len(parts) < 1 {
		return 0, fmt.Errorf("invalid boottime format")
	}
	secPart := strings.TrimPrefix(parts[0], "{ sec = ")
	sec, err := strconv.ParseInt(secPart, 10, 64)
	if err != nil {
		return 0, err
	}

	return float64(time.Now().Unix() - sec), nil
}

func getDiskUsage(path string) (float64, error) {
	var stat syscall.Statfs_t
	err := syscall.Statfs(path, &stat)
	if err != nil {
		return 0, err
	}

	total := stat.Blocks * uint64(stat.Bsize)
	free := stat.Bfree * uint64(stat.Bsize)
	used := total - free

	if total == 0 {
		return 0, nil
	}

	return float64(used) / float64(total) * 100, nil
}

func (c *Collector) GetTopProcesses(n int) ([]Process, error) {
	// Use ps to get top memory consuming processes
	cmd := exec.Command("ps", "-Aco", "pid,rss,comm", "-m")
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		return nil, err
	}

	lines := strings.Split(out.String(), "\n")
	var processes []Process
	for i, line := range lines {
		if i == 0 || line == "" {
			continue // skip header and empty lines
		}
		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}

		pid, _ := strconv.Atoi(fields[0])
		rss, _ := strconv.ParseUint(fields[1], 10, 64) // RSS is in KB on macOS ps
		name := strings.Join(fields[2:], " ")

		processes = append(processes, Process{
			PID:    pid,
			Memory: rss,
			Name:   name,
		})
	}

	sort.Slice(processes, func(i, j int) bool {
		return processes[i].Memory > processes[j].Memory
	})

	if len(processes) > n {
		processes = processes[:n]
	}

	return processes, nil
}
