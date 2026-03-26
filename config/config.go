package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// getDefaultConfig loads the default configuration from config-sample.yml
func getDefaultConfig(configDir string) *Config {
	// Try to find config-sample.yml relative to the working directory
	defaultConfigPath := "config-sample.yml"

	data, err := os.ReadFile(defaultConfigPath)
	if err != nil {
		// Fallback to hardcoded defaults if config-sample.yml is not found
		return &Config{
			PollInterval:         15,
			CPUThresholdPercent:  85,
			CPURecoveryPercent:   70,
			CPUSustainSeconds:    300,
			MemThresholdPercent:  90,
			MemRecoveryPercent:   80,
			MemSustainSeconds:    180,
			SwapThresholdPercent: 25,
			SwapRecoveryPercent:  10,
			SwapSustainSeconds:   180,
			DiskThresholdPercent: 95,
			DiskRecoveryPercent:  90,
			AlertCooldownSeconds: 1800,
			TopProcessCount:      5,
			DBPath:               filepath.Join(configDir, "telemon.db"),
		}
	}

	var c Config
	if err := yaml.Unmarshal(data, &c); err != nil {
		// Fallback to hardcoded defaults if unmarshaling fails
		return &Config{
			PollInterval:         15,
			CPUThresholdPercent:  85,
			CPURecoveryPercent:   70,
			CPUSustainSeconds:    300,
			MemThresholdPercent:  90,
			MemRecoveryPercent:   80,
			MemSustainSeconds:    180,
			SwapThresholdPercent: 25,
			SwapRecoveryPercent:  10,
			SwapSustainSeconds:   180,
			DiskThresholdPercent: 95,
			DiskRecoveryPercent:  90,
			AlertCooldownSeconds: 1800,
			TopProcessCount:      5,
			DBPath:               filepath.Join(configDir, "telemon.db"),
		}
	}

	// Ensure DBPath uses the config directory
	if c.DBPath == "" {
		c.DBPath = filepath.Join(configDir, "telemon.db")
	}

	return &c
}

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

	AlertCooldownSeconds int    `yaml:"alert_cooldown_seconds"`
	TopProcessCount      int    `yaml:"top_process_count"`
	DBPath               string `yaml:"db_path"`
}

func (c *Config) Validate() error {
	if c.PollInterval < 1 {
		return fmt.Errorf("poll_interval_seconds must be at least 1")
	}
	return nil
}

func GetConfigDir() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user config directory: %w", err)
	}

	appDir := filepath.Join(configDir, "tg-system-monitor")
	if err := os.MkdirAll(appDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create app config directory: %w", err)
	}

	return appDir, nil
}

var GetConfigPath = func() (string, error) {
	appDir, err := GetConfigDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(appDir, "config.yml"), nil
}

func Load() (*Config, error) {
	configDir, err := GetConfigDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get user config directory: %w", err)
	}

	path, err := GetConfigPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			// Return default config from config-sample.yml if file doesn't exist
			return getDefaultConfig(configDir), nil
		}
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var c Config
	if err := yaml.Unmarshal(data, &c); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	if err := c.Validate(); err != nil {
		return nil, err
	}

	return &c, nil
}

func (c *Config) Save() error {
	path, err := GetConfigPath()
	if err != nil {
		return err
	}

	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}
