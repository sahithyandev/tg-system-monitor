package config

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

var DefaultConfigYAML string

// getDefaultConfig loads the default configuration from embedded default-config.yml
func getDefaultConfig(configDir string) *Config {
	// Use embedded default config
	data := []byte(DefaultConfigYAML)

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
			DBPath:               filepath.Join(configDir, "tgsm1.db"),
		}
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

// mergeConfigs merges user configuration on top of default configuration
// User config values take priority over default values
func mergeConfigs(defaultConfig, userConfig *Config) *Config {
	if userConfig == nil {
		return defaultConfig
	}

	merged := *defaultConfig // Copy default config

	// Override with user values if they are set
	if userConfig.BotToken != "" {
		merged.BotToken = userConfig.BotToken
	}
	if userConfig.JoinPasswordHash != "" {
		merged.JoinPasswordHash = userConfig.JoinPasswordHash
	}
	if userConfig.HostnameOverride != "" {
		merged.HostnameOverride = userConfig.HostnameOverride
	}
	if userConfig.PollInterval != 0 {
		merged.PollInterval = userConfig.PollInterval
	}
	if userConfig.CPUThresholdPercent != 0 {
		merged.CPUThresholdPercent = userConfig.CPUThresholdPercent
	}
	if userConfig.CPURecoveryPercent != 0 {
		merged.CPURecoveryPercent = userConfig.CPURecoveryPercent
	}
	if userConfig.CPUSustainSeconds != 0 {
		merged.CPUSustainSeconds = userConfig.CPUSustainSeconds
	}
	if userConfig.MemThresholdPercent != 0 {
		merged.MemThresholdPercent = userConfig.MemThresholdPercent
	}
	if userConfig.MemRecoveryPercent != 0 {
		merged.MemRecoveryPercent = userConfig.MemRecoveryPercent
	}
	if userConfig.MemSustainSeconds != 0 {
		merged.MemSustainSeconds = userConfig.MemSustainSeconds
	}
	if userConfig.SwapThresholdPercent != 0 {
		merged.SwapThresholdPercent = userConfig.SwapThresholdPercent
	}
	if userConfig.SwapRecoveryPercent != 0 {
		merged.SwapRecoveryPercent = userConfig.SwapRecoveryPercent
	}
	if userConfig.SwapSustainSeconds != 0 {
		merged.SwapSustainSeconds = userConfig.SwapSustainSeconds
	}
	if userConfig.DiskThresholdPercent != 0 {
		merged.DiskThresholdPercent = userConfig.DiskThresholdPercent
	}
	if userConfig.DiskRecoveryPercent != 0 {
		merged.DiskRecoveryPercent = userConfig.DiskRecoveryPercent
	}
	if userConfig.AlertCooldownSeconds != 0 {
		merged.AlertCooldownSeconds = userConfig.AlertCooldownSeconds
	}
	if userConfig.TopProcessCount != 0 {
		merged.TopProcessCount = userConfig.TopProcessCount
	}
	if userConfig.DBPath != "" {
		merged.DBPath = userConfig.DBPath
	}

	return &merged
}

func Load() (*Config, error) {
	configDir, err := GetConfigDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get user config directory: %w", err)
	}

	// Always load default config first
	defaultConfig := getDefaultConfig(configDir)
	fmt.Println("defaultConfig", defaultConfig)

	path, err := GetConfigPath()
	if err != nil {
		return nil, err
	}
	fmt.Println("path", path)

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			// Return default config if user config file doesn't exist
			return defaultConfig, nil
		}
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var userConfig Config
	if err := yaml.Unmarshal(data, &userConfig); err != nil {
		return nil, fmt.Errorf("failed to unmarshal user config: %w", err)
	}
	fmt.Println("userConfig", userConfig)

	// Merge user config with defaults
	mergedConfig := mergeConfigs(defaultConfig, &userConfig)

	if err := mergedConfig.Validate(); err != nil {
		return nil, err
	}

	return mergedConfig, nil
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
