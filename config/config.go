package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"tg-system-monitor/message"
	msg "tg-system-monitor/message"

	"gopkg.in/yaml.v3"
)

// expandPath expands ~ to the user's home directory
func expandPath(path string) string {
	if path == "~" {
		home, err := os.UserHomeDir()
		if err != nil {
			// If we can't get the home directory, return the path as-is
			return path
		}
		return home
	}
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			// If we can't get the home directory, return the path as-is
			return path
		}
		return filepath.Join(home, path[2:])
	}
	return path
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

	Load1Warning   float64 `yaml:"load1_warning"`
	Load1Critical  float64 `yaml:"load1_critical"`
	Load5Warning   float64 `yaml:"load5_warning"`
	Load5Critical  float64 `yaml:"load5_critical"`
	Load15Warning  float64 `yaml:"load15_warning"`
	Load15Critical float64 `yaml:"load15_critical"`
	Hysteresis     float64 `yaml:"hysteresis"`

	AlertCooldownSeconds int    `yaml:"alert_cooldown_seconds"`
	TopProcessCount      int    `yaml:"top_process_count"`
	DBPath               string `yaml:"db_path"`
}

func (c *Config) PrintThresholds() {
	fmt.Printf("%s\n", msg.StatusTemplate("System Monitor Configuration", "Active", "Thresholds and settings loaded successfully"))
	fmt.Println("==========================")

	fmt.Printf("CPU     | Warning: %.1f%% | Critical: %.1f%% | Sustain: %d seconds\n",
		c.CPURecoveryPercent, c.CPUThresholdPercent, c.CPUSustainSeconds)
	fmt.Printf("Memory  | Warning: %.1f%% | Critical: %.1f%% | Sustain: %d seconds\n",
		c.MemRecoveryPercent, c.MemThresholdPercent, c.MemSustainSeconds)
	fmt.Printf("Swap    | Warning: %.1f%% | Critical: %.1f%% | Sustain: %d seconds\n",
		c.SwapRecoveryPercent, c.SwapThresholdPercent, c.SwapSustainSeconds)
	fmt.Printf("Disk    | Warning: %.1f%% | Critical: %.1f%% | Sustain: - seconds\n",
		c.DiskRecoveryPercent, c.DiskThresholdPercent)
	fmt.Printf("Load1   | Warning: %.1f | Critical: %.1f\n",
		c.Load1Warning, c.Load1Critical)
	fmt.Printf("Load5   | Warning: %.1f | Critical: %.1f\n",
		c.Load5Warning, c.Load5Critical)
	fmt.Printf("Load15  | Warning: %.1f | Critical: %.1f\n",
		c.Load15Warning, c.Load15Critical)
	fmt.Printf("Hysteresis: %.1f%%\n", c.Hysteresis)
	fmt.Println()
}

func (c *Config) Validate() error {
	if c.PollInterval < 1 {
		fmt.Fprintln(os.Stderr, "poll_interval_seconds must be at least 1, using default 15s")
		c.PollInterval = 15
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
	if userConfig.Load1Warning != 0 {
		merged.Load1Warning = userConfig.Load1Warning
	}
	if userConfig.Load1Critical != 0 {
		merged.Load1Critical = userConfig.Load1Critical
	}
	if userConfig.Load5Warning != 0 {
		merged.Load5Warning = userConfig.Load5Warning
	}
	if userConfig.Load5Critical != 0 {
		merged.Load5Critical = userConfig.Load5Critical
	}
	if userConfig.Load15Warning != 0 {
		merged.Load15Warning = userConfig.Load15Warning
	}
	if userConfig.Load15Critical != 0 {
		merged.Load15Critical = userConfig.Load15Critical
	}
	if userConfig.Hysteresis != 0 {
		merged.Hysteresis = userConfig.Hysteresis
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

	path, err := GetConfigPath()
	if err != nil {
		return nil, err
	}

	fmt.Printf("%s\n", message.LogStarted(message.ComponentConfig, fmt.Sprintf("using config file: %s", path)))

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			// Expand ~ in DBPath for default config
			defaultConfig.DBPath = expandPath(defaultConfig.DBPath)
			// Return default config if user config file doesn't exist
			return defaultConfig, nil
		}
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var userConfig Config
	if err := yaml.Unmarshal(data, &userConfig); err != nil {
		return nil, fmt.Errorf("failed to unmarshal user config: %w", err)
	}

	// Merge user config with defaults
	mergedConfig := mergeConfigs(defaultConfig, &userConfig)

	// Expand ~ in DBPath to user's home directory
	mergedConfig.DBPath = expandPath(mergedConfig.DBPath)

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

// UpdatePasswordHash updates only the join_password_hash field in the user config file
// without affecting other configuration fields
func UpdatePasswordHash(hash string) error {
	path, err := GetConfigPath()
	if err != nil {
		return fmt.Errorf("failed to get config path: %w", err)
	}

	var userConfig map[string]interface{}

	// Try to read existing user config
	data, err := os.ReadFile(path)
	if err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("failed to read config file: %w", err)
		}
		// Config file doesn't exist, create empty config
		userConfig = make(map[string]interface{})
	} else {
		// Parse existing config
		if err := yaml.Unmarshal(data, &userConfig); err != nil {
			return fmt.Errorf("failed to unmarshal existing config: %w", err)
		}
	}

	// Update only the password hash field
	userConfig["join_password_hash"] = hash

	// Marshal and write back
	data, err = yaml.Marshal(userConfig)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}
