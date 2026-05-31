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

// Config represents the configuration structure
type Config struct {
	BotToken         string `yaml:"bot_token"`
	JoinPasswordHash string `yaml:"join_password_hash"`
	HostnameOverride string `yaml:"hostname_override"`
	PollInterval     int    `yaml:"poll_interval_seconds"`

	CPUThresholdPercent float64 `yaml:"cpu_threshold_percent"`
	CPURecoveryPercent  float64 `yaml:"cpu_recovery_percent"`
	CPUSustainSeconds   int     `yaml:"cpu_sustain_seconds"`

	MemThresholdPercent float64 `yaml:"mem_threshold_percent"`
	MemRecoveryPercent  float64 `yaml:"mem_recovery_percent"`
	MemSustainSeconds   int     `yaml:"mem_sustain_seconds"`

	SwapThresholdPercent float64 `yaml:"swap_threshold_percent"`
	SwapRecoveryPercent  float64 `yaml:"swap_recovery_percent"`
	SwapSustainSeconds   int     `yaml:"swap_sustain_seconds"`

	DiskThresholdPercent float64 `yaml:"disk_threshold_percent"`
	DiskRecoveryPercent  float64 `yaml:"disk_recovery_percent"`

	Load1Warning   float64 `yaml:"load1_warning"`
	Load1Critical  float64 `yaml:"load1_critical"`
	Load5Warning   float64 `yaml:"load5_warning"`
	Load5Critical  float64 `yaml:"load5_critical"`
	Load15Warning  float64 `yaml:"load15_warning"`
	Load15Critical float64 `yaml:"load15_critical"`

	Hysteresis           float64 `yaml:"hysteresis"`
	AlertCooldownSeconds int     `yaml:"alert_cooldown_seconds"`
	TopProcessCount      int     `yaml:"top_process_count"`
	DBPath               string         `yaml:"db_path"`
	DataRetentionDays    int            `yaml:"data_retention_days"`
	Volumes              []VolumeConfig `yaml:"volumes"`
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
	var builder strings.Builder

	builder.WriteString("package config\n\n")
	builder.WriteString("import \"path/filepath\"\n\n")
	builder.WriteString("// getDefaultConfig returns hardcoded default configuration values\n")
	builder.WriteString("// This file is auto-generated from default-config.yml\n")
	builder.WriteString("// DO NOT EDIT MANUALLY - Run \"go run tools/generate_defaults.go\" to regenerate\n")
	builder.WriteString("func getDefaultConfig(configDir string) *Config {\n")
	builder.WriteString("\treturn &Config{\n")
	builder.WriteString("\t\tBotToken:             ")
	builder.WriteString(fmt.Sprintf("%q,\n", config.BotToken))

	builder.WriteString("\t\tJoinPasswordHash:     ")
	builder.WriteString(fmt.Sprintf("%q,\n", config.JoinPasswordHash))

	builder.WriteString("\t\tHostnameOverride:     ")
	builder.WriteString(fmt.Sprintf("%q,\n", config.HostnameOverride))

	builder.WriteString("\t\tPollInterval:         ")
	builder.WriteString(fmt.Sprintf("%d,\n", config.PollInterval))

	builder.WriteString("\t\tCPUThresholdPercent:  ")
	builder.WriteString(fmt.Sprintf("%.1f,\n", config.CPUThresholdPercent))

	builder.WriteString("\t\tCPURecoveryPercent:   ")
	builder.WriteString(fmt.Sprintf("%.1f,\n", config.CPURecoveryPercent))

	builder.WriteString("\t\tCPUSustainSeconds:    ")
	builder.WriteString(fmt.Sprintf("%d,\n", config.CPUSustainSeconds))

	builder.WriteString("\t\tMemThresholdPercent:  ")
	builder.WriteString(fmt.Sprintf("%.1f,\n", config.MemThresholdPercent))

	builder.WriteString("\t\tMemRecoveryPercent:   ")
	builder.WriteString(fmt.Sprintf("%.1f,\n", config.MemRecoveryPercent))

	builder.WriteString("\t\tMemSustainSeconds:    ")
	builder.WriteString(fmt.Sprintf("%d,\n", config.MemSustainSeconds))

	builder.WriteString("\t\tSwapThresholdPercent: ")
	builder.WriteString(fmt.Sprintf("%.1f,\n", config.SwapThresholdPercent))

	builder.WriteString("\t\tSwapRecoveryPercent:  ")
	builder.WriteString(fmt.Sprintf("%.1f,\n", config.SwapRecoveryPercent))

	builder.WriteString("\t\tSwapSustainSeconds:   ")
	builder.WriteString(fmt.Sprintf("%d,\n", config.SwapSustainSeconds))

	builder.WriteString("\t\tDiskThresholdPercent: ")
	builder.WriteString(fmt.Sprintf("%.1f,\n", config.DiskThresholdPercent))

	builder.WriteString("\t\tDiskRecoveryPercent:  ")
	builder.WriteString(fmt.Sprintf("%.1f,\n", config.DiskRecoveryPercent))

	builder.WriteString("\t\tLoad1Warning:         ")
	builder.WriteString(fmt.Sprintf("%.1f,\n", config.Load1Warning))

	builder.WriteString("\t\tLoad1Critical:        ")
	builder.WriteString(fmt.Sprintf("%.1f,\n", config.Load1Critical))

	builder.WriteString("\t\tLoad5Warning:         ")
	builder.WriteString(fmt.Sprintf("%.1f,\n", config.Load5Warning))

	builder.WriteString("\t\tLoad5Critical:        ")
	builder.WriteString(fmt.Sprintf("%.1f,\n", config.Load5Critical))

	builder.WriteString("\t\tLoad15Warning:        ")
	builder.WriteString(fmt.Sprintf("%.1f,\n", config.Load15Warning))

	builder.WriteString("\t\tLoad15Critical:       ")
	builder.WriteString(fmt.Sprintf("%.1f,\n", config.Load15Critical))

	builder.WriteString("\t\tHysteresis:           ")
	builder.WriteString(fmt.Sprintf("%.1f,\n", config.Hysteresis))

	builder.WriteString("\t\tAlertCooldownSeconds: ")
	builder.WriteString(fmt.Sprintf("%d,\n", config.AlertCooldownSeconds))

	builder.WriteString("\t\tTopProcessCount:      ")
	builder.WriteString(fmt.Sprintf("%d,\n", config.TopProcessCount))

	builder.WriteString("\t\tDBPath:               ")
	builder.WriteString("filepath.Join(configDir, \"tgsm.db\"),\n")

	builder.WriteString("\t\tDataRetentionDays:    ")
	builder.WriteString(fmt.Sprintf("%d,\n", config.DataRetentionDays))

	builder.WriteString("\t}\n")
	builder.WriteString("}\n")

	// Write to defaults.go
	err := os.MkdirAll("config", 0755)
	if err != nil {
		fmt.Printf("Error creating config directory: %v\n", err)
		os.Exit(1)
	}

	err = os.WriteFile("config/defaults.go", []byte(builder.String()), 0644)
	if err != nil {
		fmt.Printf("Error writing defaults.go: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Successfully generated config/defaults.go from default-config.yml")
}
