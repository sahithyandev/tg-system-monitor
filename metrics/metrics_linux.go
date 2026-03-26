//go:build linux
// +build linux

package metrics

import (
	"bufio"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"
)

type Collector struct {
	prevCPUTotal uint64
	prevCPUIdle  uint64
}

func NewCollector() *Collector {
	return &Collector{}
}

func (c *Collector) Collect() (*MetricSample, error) {
	sample := &MetricSample{
		Timestamp: time.Now(),
	}

	var err error

	// CPU
	sample.CPUPercent, err = c.getCPUUsage()
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

func (c *Collector) getCPUUsage() (float64, error) {
	file, err := os.Open("/proc/stat")
	if err != nil {
		return 0, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	if !scanner.Scan() {
		return 0, fmt.Errorf("empty /proc/stat")
	}

	line := scanner.Text()
	fields := strings.Fields(line)
	if len(fields) < 5 || fields[0] != "cpu" {
		return 0, fmt.Errorf("invalid /proc/stat format")
	}

	var total uint64
	var idle uint64
	for i, field := range fields[1:] {
		val, err := strconv.ParseUint(field, 10, 64)
		if err != nil {
			return 0, err
		}
		total += val
		if i == 3 { // 'idle' is the 4th field after 'cpu'
			idle = val
		}
	}

	deltaTotal := total - c.prevCPUTotal
	deltaIdle := idle - c.prevCPUIdle

	c.prevCPUTotal = total
	c.prevCPUIdle = idle

	if deltaTotal == 0 {
		return 0, nil
	}

	cpuPercent := float64(deltaTotal-deltaIdle) / float64(deltaTotal) * 100
	return cpuPercent, nil
}

func getMemoryUsage() (float64, float64, error) {
	file, err := os.Open("/proc/meminfo")
	if err != nil {
		return 0, 0, err
	}
	defer file.Close()

	var memTotal, memAvailable, swapTotal, swapFree uint64
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		val, err := strconv.ParseUint(fields[1], 10, 64)
		if err != nil {
			continue
		}

		switch fields[0] {
		case "MemTotal:":
			memTotal = val
		case "MemAvailable:":
			memAvailable = val
		case "SwapTotal:":
			swapTotal = val
		case "SwapFree:":
			swapFree = val
		}
	}

	memPercent := 0.0
	if memTotal > 0 {
		memPercent = float64(memTotal-memAvailable) / float64(memTotal) * 100
	}

	swapPercent := 0.0
	if swapTotal > 0 {
		swapPercent = float64(swapTotal-swapFree) / float64(swapTotal) * 100
	}

	return memPercent, swapPercent, nil
}

func getLoadAvg() (float64, float64, float64, error) {
	data, err := os.ReadFile("/proc/loadavg")
	if err != nil {
		return 0, 0, 0, err
	}

	fields := strings.Fields(string(data))
	if len(fields) < 3 {
		return 0, 0, 0, fmt.Errorf("invalid /proc/loadavg format")
	}

	l1, err := strconv.ParseFloat(fields[0], 64)
	if err != nil {
		return 0, 0, 0, err
	}
	l5, err := strconv.ParseFloat(fields[1], 64)
	if err != nil {
		return 0, 0, 0, err
	}
	l15, err := strconv.ParseFloat(fields[2], 64)
	if err != nil {
		return 0, 0, 0, err
	}

	return l1, l5, l15, nil
}

func getUptime() (float64, error) {
	data, err := os.ReadFile("/proc/uptime")
	if err != nil {
		return 0, err
	}

	fields := strings.Fields(string(data))
	if len(fields) < 1 {
		return 0, fmt.Errorf("invalid /proc/uptime format")
	}

	uptime, err := strconv.ParseFloat(fields[0], 64)
	if err != nil {
		return 0, err
	}

	return uptime, nil
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
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return nil, err
	}

	var processes []Process
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		pid, err := strconv.Atoi(entry.Name())
		if err != nil {
			continue
		}

		p, err := c.getProcessInfo(pid)
		if err != nil {
			continue
		}
		processes = append(processes, p)
	}

	// Sort by Memory (RSS) descending
	sort.Slice(processes, func(i, j int) bool {
		return processes[i].Memory > processes[j].Memory
	})

	if len(processes) > n {
		processes = processes[:n]
	}

	return processes, nil
}

func (c *Collector) getProcessInfo(pid int) (Process, error) {
	statusFile := fmt.Sprintf("/proc/%d/status", pid)
	file, err := os.Open(statusFile)
	if err != nil {
		return Process{}, err
	}
	defer file.Close()

	var p Process
	p.PID = pid
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "Name:") {
			p.Name = strings.TrimSpace(strings.TrimPrefix(line, "Name:"))
		} else if strings.HasPrefix(line, "VmRSS:") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				val, _ := strconv.ParseUint(fields[1], 10, 64)
				p.Memory = val
			}
		}
	}

	return p, nil
}
