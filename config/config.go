package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

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

// VolumeConfig represents a single extra volume to monitor.
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

// DiskConfig holds disk alert thresholds and extra volumes to monitor.
type DiskConfig struct {
	ThresholdPercent float64        `yaml:"threshold_percent"`
	RecoveryPercent  float64        `yaml:"recovery_percent"`
	Volumes          []VolumeConfig `yaml:"volumes"`
}

// MonitorsConfig holds all per-metric monitoring thresholds.
type MonitorsConfig struct {
	CPU    MetricThreshold `yaml:"cpu"`
	Memory MetricThreshold `yaml:"memory"`
	Swap   MetricThreshold `yaml:"swap"`
	Disk   DiskConfig      `yaml:"disk"`
	Load   LoadConfig      `yaml:"load"`
}

// Config is the canonical application configuration.
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

	MetricsAPIAddr string `yaml:"metrics_api_addr"`

	Monitors MonitorsConfig `yaml:"monitors"`
}


func (c *Config) PrintThresholds() {
	fmt.Printf("%s\n", msg.StatusTemplate("System Monitor Configuration", "Active", "Thresholds and settings loaded successfully"))
	fmt.Println("==========================")

	m := c.Monitors
	fmt.Printf("CPU     | Warning: %.1f%% | Critical: %.1f%% | Sustain: %d seconds\n",
		m.CPU.ThresholdPercent, m.CPU.RecoveryPercent, m.CPU.SustainSeconds)
	fmt.Printf("Memory  | Warning: %.1f%% | Critical: %.1f%% | Sustain: %d seconds\n",
		m.Memory.ThresholdPercent, m.Memory.RecoveryPercent, m.Memory.SustainSeconds)
	fmt.Printf("Swap    | Warning: %.1f%% | Critical: %.1f%% | Sustain: %d seconds\n",
		m.Swap.ThresholdPercent, m.Swap.RecoveryPercent, m.Swap.SustainSeconds)
	fmt.Printf("Disk    | Warning: %.1f%% | Critical: %.1f%% | Sustain: - seconds\n",
		m.Disk.ThresholdPercent, m.Disk.RecoveryPercent)
	fmt.Printf("Load1   | Warning: %.1f | Critical: %.1f\n",
		m.Load.Load1.Warning, m.Load.Load1.Critical)
	fmt.Printf("Load5   | Warning: %.1f | Critical: %.1f\n",
		m.Load.Load5.Warning, m.Load.Load5.Critical)
	fmt.Printf("Load15  | Warning: %.1f | Critical: %.1f\n",
		m.Load.Load15.Warning, m.Load.Load15.Critical)
	fmt.Printf("Hysteresis: %.1f%%\n", c.Hysteresis)
	if len(m.Disk.Volumes) > 0 {
		fmt.Println("Additional volumes:")
		for _, v := range m.Disk.Volumes {
			threshold := v.ThresholdPercent
			if threshold == 0 {
				threshold = m.Disk.ThresholdPercent
			}
			recovery := v.RecoveryPercent
			if recovery == 0 {
				recovery = m.Disk.RecoveryPercent
			}
			fmt.Printf("  %-20s | Warning: %.1f%% | Critical: %.1f%%\n", v.Path, threshold, recovery)
		}
	}
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

// mergeMonitors merges non-zero user values on top of default values for monitors.
func mergeMonitors(dst, src *MonitorsConfig) {
	if src.CPU.ThresholdPercent != 0 {
		dst.CPU.ThresholdPercent = src.CPU.ThresholdPercent
	}
	if src.CPU.RecoveryPercent != 0 {
		dst.CPU.RecoveryPercent = src.CPU.RecoveryPercent
	}
	if src.CPU.SustainSeconds != 0 {
		dst.CPU.SustainSeconds = src.CPU.SustainSeconds
	}
	if src.Memory.ThresholdPercent != 0 {
		dst.Memory.ThresholdPercent = src.Memory.ThresholdPercent
	}
	if src.Memory.RecoveryPercent != 0 {
		dst.Memory.RecoveryPercent = src.Memory.RecoveryPercent
	}
	if src.Memory.SustainSeconds != 0 {
		dst.Memory.SustainSeconds = src.Memory.SustainSeconds
	}
	if src.Swap.ThresholdPercent != 0 {
		dst.Swap.ThresholdPercent = src.Swap.ThresholdPercent
	}
	if src.Swap.RecoveryPercent != 0 {
		dst.Swap.RecoveryPercent = src.Swap.RecoveryPercent
	}
	if src.Swap.SustainSeconds != 0 {
		dst.Swap.SustainSeconds = src.Swap.SustainSeconds
	}
	if src.Disk.ThresholdPercent != 0 {
		dst.Disk.ThresholdPercent = src.Disk.ThresholdPercent
	}
	if src.Disk.RecoveryPercent != 0 {
		dst.Disk.RecoveryPercent = src.Disk.RecoveryPercent
	}
	if len(src.Disk.Volumes) > 0 {
		dst.Disk.Volumes = src.Disk.Volumes
	}
	if src.Load.Load1.Warning != 0 {
		dst.Load.Load1.Warning = src.Load.Load1.Warning
	}
	if src.Load.Load1.Critical != 0 {
		dst.Load.Load1.Critical = src.Load.Load1.Critical
	}
	if src.Load.Load5.Warning != 0 {
		dst.Load.Load5.Warning = src.Load.Load5.Warning
	}
	if src.Load.Load5.Critical != 0 {
		dst.Load.Load5.Critical = src.Load.Load5.Critical
	}
	if src.Load.Load15.Warning != 0 {
		dst.Load.Load15.Warning = src.Load.Load15.Warning
	}
	if src.Load.Load15.Critical != 0 {
		dst.Load.Load15.Critical = src.Load.Load15.Critical
	}
}

// mergeConfigs merges user configuration on top of default configuration.
// User config values take priority over default values.
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
	if userConfig.AlertCooldownSeconds != 0 {
		merged.AlertCooldownSeconds = userConfig.AlertCooldownSeconds
	}
	if userConfig.TopProcessCount != 0 {
		merged.TopProcessCount = userConfig.TopProcessCount
	}
	if userConfig.DBPath != "" {
		merged.DBPath = userConfig.DBPath
	}
	if userConfig.DataRetentionDays != 0 {
		merged.DataRetentionDays = userConfig.DataRetentionDays
	}
	if userConfig.Hysteresis != 0 {
		merged.Hysteresis = userConfig.Hysteresis
	}
	if userConfig.MetricsAPIAddr != "" {
		merged.MetricsAPIAddr = userConfig.MetricsAPIAddr
	}

	mergeMonitors(&merged.Monitors, &userConfig.Monitors)

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

	fmt.Printf("%s\n", msg.LogStarted(msg.ComponentConfig, fmt.Sprintf("using config file: %s", path)))

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
