//go:build ignore

package main

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

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
	DBPath               string  `yaml:"db_path"`
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

	builder.WriteString(`package config

import "path/filepath"

// getDefaultConfig returns hardcoded default configuration values
// This file is auto-generated from default-config.yml
// DO NOT EDIT MANUALLY - Run "go run tools/generate_defaults.go" to regenerate
func getDefaultConfig(configDir string) *Config {
	return &Config{
		PollInterval:         `)
	builder.WriteString(fmt.Sprintf("%d,\n", config.PollInterval))

	builder.WriteString(`		CPUThresholdPercent:  `)
	builder.WriteString(fmt.Sprintf("%.1f,\n", config.CPUThresholdPercent))

	builder.WriteString(`		CPURecoveryPercent:   `)
	builder.WriteString(fmt.Sprintf("%.1f,\n", config.CPURecoveryPercent))

	builder.WriteString(`		CPUSustainSeconds:    `)
	builder.WriteString(fmt.Sprintf("%d,\n", config.CPUSustainSeconds))

	builder.WriteString(`		MemThresholdPercent:  `)
	builder.WriteString(fmt.Sprintf("%.1f,\n", config.MemThresholdPercent))

	builder.WriteString(`		MemRecoveryPercent:   `)
	builder.WriteString(fmt.Sprintf("%.1f,\n", config.MemRecoveryPercent))

	builder.WriteString(`		MemSustainSeconds:    `)
	builder.WriteString(fmt.Sprintf("%d,\n", config.MemSustainSeconds))

	builder.WriteString(`		SwapThresholdPercent: `)
	builder.WriteString(fmt.Sprintf("%.1f,\n", config.SwapThresholdPercent))

	builder.WriteString(`		SwapRecoveryPercent:  `)
	builder.WriteString(fmt.Sprintf("%.1f,\n", config.SwapRecoveryPercent))

	builder.WriteString(`		SwapSustainSeconds:   `)
	builder.WriteString(fmt.Sprintf("%d,\n", config.SwapSustainSeconds))

	builder.WriteString(`		DiskThresholdPercent: `)
	builder.WriteString(fmt.Sprintf("%.1f,\n", config.DiskThresholdPercent))

	builder.WriteString(`		DiskRecoveryPercent:  `)
	builder.WriteString(fmt.Sprintf("%.1f,\n", config.DiskRecoveryPercent))

	builder.WriteString(`		Load1Warning:         `)
	builder.WriteString(fmt.Sprintf("%.1f,\n", config.Load1Warning))

	builder.WriteString(`		Load1Critical:        `)
	builder.WriteString(fmt.Sprintf("%.1f,\n", config.Load1Critical))

	builder.WriteString(`		Load5Warning:         `)
	builder.WriteString(fmt.Sprintf("%.1f,\n", config.Load5Warning))

	builder.WriteString(`		Load5Critical:        `)
	builder.WriteString(fmt.Sprintf("%.1f,\n", config.Load5Critical))

	builder.WriteString(`		Load15Warning:        `)
	builder.WriteString(fmt.Sprintf("%.1f,\n", config.Load15Warning))

	builder.WriteString(`		Load15Critical:       `)
	builder.WriteString(fmt.Sprintf("%.1f,\n", config.Load15Critical))

	builder.WriteString(`		Hysteresis:           `)
	builder.WriteString(fmt.Sprintf("%.1f,\n", config.Hysteresis))

	builder.WriteString(`		AlertCooldownSeconds: `)
	builder.WriteString(fmt.Sprintf("%d,\n", config.AlertCooldownSeconds))

	builder.WriteString(`		TopProcessCount:      `)
	builder.WriteString(fmt.Sprintf("%d,\n", config.TopProcessCount))

	builder.WriteString(`		DBPath:               filepath.Join(configDir, "tgsm.db"),
	}
}
`)

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
