//go:build ignore

package main

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// VolumeConfig represents a single monitored volume
type VolumeConfig struct {
	Path             string  `yaml:"path"`
	ThresholdPercent float64 `yaml:"threshold_percent"`
	RecoveryPercent  float64 `yaml:"recovery_percent"`
}

// MetricThreshold holds alert thresholds for a single metric type.
type MetricThreshold struct {
	ThresholdPercent float64 `yaml:"threshold_percent"`
	RecoveryPercent  float64 `yaml:"recovery_percent"`
	SustainSeconds   int     `yaml:"sustain_seconds"`
}

// DiskConfig holds disk alert thresholds and extra volumes to monitor.
type DiskConfig struct {
	ThresholdPercent float64        `yaml:"threshold_percent"`
	RecoveryPercent  float64        `yaml:"recovery_percent"`
	Volumes          []VolumeConfig `yaml:"volumes"`
}

// LoadLevel holds warning/critical levels for a single load-average window.
type LoadLevel struct {
	Warning  float64 `yaml:"warning"`
	Critical float64 `yaml:"critical"`
}

// LoadConfig groups load-average thresholds for all three windows.
type LoadConfig struct {
	Load1  LoadLevel `yaml:"load1"`
	Load5  LoadLevel `yaml:"load5"`
	Load15 LoadLevel `yaml:"load15"`
}

// MonitorsConfig holds all per-metric monitoring thresholds.
type MonitorsConfig struct {
	CPU    MetricThreshold `yaml:"cpu"`
	Memory MetricThreshold `yaml:"memory"`
	Swap   MetricThreshold `yaml:"swap"`
	Disk   DiskConfig      `yaml:"disk"`
	Load   LoadConfig      `yaml:"load"`
}

// Config represents the configuration structure
type Config struct {
	BotToken         string `yaml:"bot_token"`
	JoinPasswordHash string `yaml:"join_password_hash"`
	HostnameOverride string `yaml:"hostname_override"`
	PollInterval     int    `yaml:"poll_interval_seconds"`

	AlertCooldownSeconds int     `yaml:"alert_cooldown_seconds"`
	TopProcessCount      int     `yaml:"top_process_count"`
	DBPath               string  `yaml:"db_path"`
	DataRetentionDays    int     `yaml:"data_retention_days"`
	Hysteresis           float64 `yaml:"hysteresis"`

	Monitors MonitorsConfig `yaml:"monitors"`
}

func main() {
	// Read default-config.yml
	data, err := os.ReadFile("default-config.yml")
	if err != nil {
		fmt.Printf("Error reading default-config.yml: %v\n", err)
		os.Exit(1)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		fmt.Printf("Error parsing default-config.yml: %v\n", err)
		os.Exit(1)
	}

	// Generate Go code with hardcoded defaults
	generateDefaultsFile(config)
}

func generateDefaultsFile(config Config) {
	var b strings.Builder

	b.WriteString("package config\n\n")
	b.WriteString("import \"path/filepath\"\n\n")
	b.WriteString("// getDefaultConfig returns hardcoded default configuration values\n")
	b.WriteString("// This file is auto-generated from default-config.yml\n")
	b.WriteString("// DO NOT EDIT MANUALLY - Run \"go run tools/generate_defaults.go\" to regenerate\n")
	b.WriteString("func getDefaultConfig(configDir string) *Config {\n")
	b.WriteString("\treturn &Config{\n")

	fmt.Fprintf(&b, "\t\tBotToken:             %q,\n", config.BotToken)
	fmt.Fprintf(&b, "\t\tJoinPasswordHash:     %q,\n", config.JoinPasswordHash)
	fmt.Fprintf(&b, "\t\tHostnameOverride:     %q,\n", config.HostnameOverride)
	fmt.Fprintf(&b, "\t\tPollInterval:         %d,\n", config.PollInterval)
	fmt.Fprintf(&b, "\t\tAlertCooldownSeconds: %d,\n", config.AlertCooldownSeconds)
	fmt.Fprintf(&b, "\t\tTopProcessCount:      %d,\n", config.TopProcessCount)
	b.WriteString("\t\tDBPath:               filepath.Join(configDir, \"tgsm.db\"),\n")
	fmt.Fprintf(&b, "\t\tDataRetentionDays:    %d,\n", config.DataRetentionDays)
	fmt.Fprintf(&b, "\t\tHysteresis:           %.1f,\n", config.Hysteresis)

	m := config.Monitors
	b.WriteString("\t\tMonitors: MonitorsConfig{\n")

	b.WriteString("\t\t\tCPU: MetricThreshold{\n")
	fmt.Fprintf(&b, "\t\t\t\tThresholdPercent: %.1f,\n", m.CPU.ThresholdPercent)
	fmt.Fprintf(&b, "\t\t\t\tRecoveryPercent:  %.1f,\n", m.CPU.RecoveryPercent)
	fmt.Fprintf(&b, "\t\t\t\tSustainSeconds:   %d,\n", m.CPU.SustainSeconds)
	b.WriteString("\t\t\t},\n")

	b.WriteString("\t\t\tMemory: MetricThreshold{\n")
	fmt.Fprintf(&b, "\t\t\t\tThresholdPercent: %.1f,\n", m.Memory.ThresholdPercent)
	fmt.Fprintf(&b, "\t\t\t\tRecoveryPercent:  %.1f,\n", m.Memory.RecoveryPercent)
	fmt.Fprintf(&b, "\t\t\t\tSustainSeconds:   %d,\n", m.Memory.SustainSeconds)
	b.WriteString("\t\t\t},\n")

	b.WriteString("\t\t\tSwap: MetricThreshold{\n")
	fmt.Fprintf(&b, "\t\t\t\tThresholdPercent: %.1f,\n", m.Swap.ThresholdPercent)
	fmt.Fprintf(&b, "\t\t\t\tRecoveryPercent:  %.1f,\n", m.Swap.RecoveryPercent)
	fmt.Fprintf(&b, "\t\t\t\tSustainSeconds:   %d,\n", m.Swap.SustainSeconds)
	b.WriteString("\t\t\t},\n")

	b.WriteString("\t\t\tDisk: DiskConfig{\n")
	fmt.Fprintf(&b, "\t\t\t\tThresholdPercent: %.1f,\n", m.Disk.ThresholdPercent)
	fmt.Fprintf(&b, "\t\t\t\tRecoveryPercent:  %.1f,\n", m.Disk.RecoveryPercent)
	b.WriteString("\t\t\t},\n")

	b.WriteString("\t\t\tLoad: LoadConfig{\n")
	fmt.Fprintf(&b, "\t\t\t\tLoad1:  LoadLevel{Warning: %.1f, Critical: %.1f},\n", m.Load.Load1.Warning, m.Load.Load1.Critical)
	fmt.Fprintf(&b, "\t\t\t\tLoad5:  LoadLevel{Warning: %.1f, Critical: %.1f},\n", m.Load.Load5.Warning, m.Load.Load5.Critical)
	fmt.Fprintf(&b, "\t\t\t\tLoad15: LoadLevel{Warning: %.1f, Critical: %.1f},\n", m.Load.Load15.Warning, m.Load.Load15.Critical)
	b.WriteString("\t\t\t},\n")

	b.WriteString("\t\t},\n")
	b.WriteString("\t}\n")
	b.WriteString("}\n")

	// Write to defaults.go
	err := os.MkdirAll("config", 0755)
	if err != nil {
		fmt.Printf("Error creating config directory: %v\n", err)
		os.Exit(1)
	}

	err = os.WriteFile("config/defaults.go", []byte(b.String()), 0644)
	if err != nil {
		fmt.Printf("Error writing defaults.go: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Successfully generated config/defaults.go from default-config.yml")
}
