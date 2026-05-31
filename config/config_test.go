package config

import (
	"os"
	"path/filepath"
	"testing"

	"gopkg.in/yaml.v3"
)

func init() {
	// Tests will use the filesystem-based config loading
}

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     Config
		wantErr bool
	}{
		{"Valid interval", Config{PollInterval: 15}, false},
		{"Minimum valid", Config{PollInterval: 1}, false},
		{"Zero interval", Config{PollInterval: 0}, false},
		{"Negative interval", Config{PollInterval: -1}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.cfg.Validate(); (err != nil) != tt.wantErr {
				t.Errorf("Config.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestConfig_LoadSave(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "config.yml")

	// Override GetConfigPath
	old := GetConfigPath
	GetConfigPath = func() (string, error) {
		return tmpFile, nil
	}
	defer func() { GetConfigPath = old }()

	// 1. Test Load default (file doesn't exist)
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.PollInterval != 15 {
		t.Errorf("expected default 15, got %d", cfg.PollInterval)
	}

	// 2. Test Save
	cfg.PollInterval = 30
	cfg.BotToken = "test_token"
	if err := cfg.Save(); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	// 3. Test Load again
	cfg2, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg2.PollInterval != 30 {
		t.Errorf("expected 30, got %d", cfg2.PollInterval)
	}
	if cfg2.BotToken != "test_token" {
		t.Errorf("expected test_token, got %s", cfg2.BotToken)
	}

	// 4. Test Load invalid config
	if err := os.WriteFile(tmpFile, []byte("poll_interval_seconds: -1"), 0644); err != nil {
		t.Fatal(err)
	}
	config, err := Load()
	if err != nil {
		t.Fatalf("expected valid fallback config, got error: %v", err)
	}
	if config.PollInterval != 15 {
		t.Errorf("expected default 15, got %d", config.PollInterval)
	}
}

// defaultTestConfig returns a fully populated Config used as the defaults baseline in
// merge tests. Uses nested MonitorsConfig to match the new struct shape.
func defaultTestConfig() *Config {
	return &Config{
		BotToken:             "default_token",
		JoinPasswordHash:     "default_hash",
		HostnameOverride:     "default_host",
		PollInterval:         15,
		AlertCooldownSeconds: 1800,
		TopProcessCount:      5,
		DBPath:               "/default/path",
		DataRetentionDays:    30,
		Hysteresis:           5.0,
		SustainSeconds:       300,
		Monitors: MonitorsConfig{
			CPU: MetricThreshold{
				ThresholdPercent: 85.0,
				RecoveryPercent:  70.0,
			},
			Memory: MetricThreshold{
				ThresholdPercent: 90.0,
				RecoveryPercent:  80.0,
			},
			Swap: MetricThreshold{
				ThresholdPercent: 25.0,
				RecoveryPercent:  10.0,
			},
			Disk: DiskConfig{
				ThresholdPercent: 95.0,
				RecoveryPercent:  90.0,
			},
			Load: LoadConfig{
				Load1:  LoadLevel{Warning: 2.0, Critical: 4.0},
				Load5:  LoadLevel{Warning: 1.5, Critical: 3.0},
				Load15: LoadLevel{Warning: 1.0, Critical: 2.0},
			},
		},
	}
}

func Test_mergeConfigs(t *testing.T) {
	defaultConfig := defaultTestConfig()

	tests := []struct {
		name       string
		userConfig *Config
		want       *Config
	}{
		{
			name:       "Nil user config",
			userConfig: nil,
			want:       defaultConfig,
		},
		{
			name:       "Empty user config",
			userConfig: &Config{},
			want:       defaultConfig,
		},
		{
			name: "Partial override — top-level scalars only",
			userConfig: &Config{
				BotToken:     "user_token",
				PollInterval: 30,
			},
			want: &Config{
				BotToken:             "user_token",
				JoinPasswordHash:     "default_hash",
				HostnameOverride:     "default_host",
				PollInterval:         30,
				AlertCooldownSeconds: 1800,
				TopProcessCount:      5,
				DBPath:               "/default/path",
				DataRetentionDays:    30,
				Hysteresis:           5.0,
				SustainSeconds:       300,
				Monitors:             defaultConfig.Monitors, // all preserved
			},
		},
		{
			name: "Partial override — nested monitors",
			userConfig: &Config{
				Monitors: MonitorsConfig{
					CPU: MetricThreshold{ThresholdPercent: 90.0},
				},
			},
			want: &Config{
				BotToken:             "default_token",
				JoinPasswordHash:     "default_hash",
				HostnameOverride:     "default_host",
				PollInterval:         15,
				AlertCooldownSeconds: 1800,
				TopProcessCount:      5,
				DBPath:               "/default/path",
				DataRetentionDays:    30,
				Hysteresis:           5.0,
				SustainSeconds:       300, // preserved
				Monitors: MonitorsConfig{
					CPU: MetricThreshold{
						ThresholdPercent: 90.0, // overridden
						RecoveryPercent:  70.0, // preserved
					},
					Memory: defaultConfig.Monitors.Memory,
					Swap:   defaultConfig.Monitors.Swap,
					Disk:   defaultConfig.Monitors.Disk,
					Load:   defaultConfig.Monitors.Load,
				},
			},
		},
		{
			name: "Full override",
			userConfig: &Config{
				BotToken:             "user_token",
				JoinPasswordHash:     "user_hash",
				HostnameOverride:     "user_host",
				PollInterval:         60,
				AlertCooldownSeconds: 3600,
				TopProcessCount:      10,
				DBPath:               "/user/path",
				DataRetentionDays:    60,
				Hysteresis:           10.0,
				SustainSeconds:       600,
				Monitors: MonitorsConfig{
					CPU:    MetricThreshold{ThresholdPercent: 90.0, RecoveryPercent: 75.0},
					Memory: MetricThreshold{ThresholdPercent: 95.0, RecoveryPercent: 85.0},
					Swap:   MetricThreshold{ThresholdPercent: 30.0, RecoveryPercent: 15.0},
					Disk:   DiskConfig{ThresholdPercent: 98.0, RecoveryPercent: 95.0},
					Load: LoadConfig{
						Load1:  LoadLevel{Warning: 3.0, Critical: 5.0},
						Load5:  LoadLevel{Warning: 2.0, Critical: 4.0},
						Load15: LoadLevel{Warning: 1.5, Critical: 2.5},
					},
				},
			},
			want: &Config{
				BotToken:             "user_token",
				JoinPasswordHash:     "user_hash",
				HostnameOverride:     "user_host",
				PollInterval:         60,
				AlertCooldownSeconds: 3600,
				TopProcessCount:      10,
				DBPath:               "/user/path",
				DataRetentionDays:    60,
				Hysteresis:           10.0,
				SustainSeconds:       600,
				Monitors: MonitorsConfig{
					CPU:    MetricThreshold{ThresholdPercent: 90.0, RecoveryPercent: 75.0},
					Memory: MetricThreshold{ThresholdPercent: 95.0, RecoveryPercent: 85.0},
					Swap:   MetricThreshold{ThresholdPercent: 30.0, RecoveryPercent: 15.0},
					Disk:   DiskConfig{ThresholdPercent: 98.0, RecoveryPercent: 95.0},
					Load: LoadConfig{
						Load1:  LoadLevel{Warning: 3.0, Critical: 5.0},
						Load5:  LoadLevel{Warning: 2.0, Critical: 4.0},
						Load15: LoadLevel{Warning: 1.5, Critical: 2.5},
					},
				},
			},
		},
		{
			name: "Undefined poll_interval_seconds uses default",
			userConfig: &Config{
				BotToken: "user_token",
				// PollInterval is zero — should fall back to default
			},
			want: &Config{
				BotToken:             "user_token",
				JoinPasswordHash:     "default_hash",
				HostnameOverride:     "default_host",
				PollInterval:         15, // default preserved
				AlertCooldownSeconds: 1800,
				TopProcessCount:      5,
				DBPath:               "/default/path",
				DataRetentionDays:    30,
				Hysteresis:           5.0,
				SustainSeconds:       300,
				Monitors:             defaultConfig.Monitors,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mergeConfigs(defaultConfig, tt.userConfig)

			if got.BotToken != tt.want.BotToken {
				t.Errorf("BotToken = %v, want %v", got.BotToken, tt.want.BotToken)
			}
			if got.JoinPasswordHash != tt.want.JoinPasswordHash {
				t.Errorf("JoinPasswordHash = %v, want %v", got.JoinPasswordHash, tt.want.JoinPasswordHash)
			}
			if got.HostnameOverride != tt.want.HostnameOverride {
				t.Errorf("HostnameOverride = %v, want %v", got.HostnameOverride, tt.want.HostnameOverride)
			}
			if got.PollInterval != tt.want.PollInterval {
				t.Errorf("PollInterval = %v, want %v", got.PollInterval, tt.want.PollInterval)
			}
			if got.AlertCooldownSeconds != tt.want.AlertCooldownSeconds {
				t.Errorf("AlertCooldownSeconds = %v, want %v", got.AlertCooldownSeconds, tt.want.AlertCooldownSeconds)
			}
			if got.TopProcessCount != tt.want.TopProcessCount {
				t.Errorf("TopProcessCount = %v, want %v", got.TopProcessCount, tt.want.TopProcessCount)
			}
			if got.DBPath != tt.want.DBPath {
				t.Errorf("DBPath = %v, want %v", got.DBPath, tt.want.DBPath)
			}
			if got.DataRetentionDays != tt.want.DataRetentionDays {
				t.Errorf("DataRetentionDays = %v, want %v", got.DataRetentionDays, tt.want.DataRetentionDays)
			}
			if got.Hysteresis != tt.want.Hysteresis {
				t.Errorf("Hysteresis = %v, want %v", got.Hysteresis, tt.want.Hysteresis)
			}
			if got.SustainSeconds != tt.want.SustainSeconds {
				t.Errorf("SustainSeconds = %v, want %v", got.SustainSeconds, tt.want.SustainSeconds)
			}
			// Monitors
			wm, gm := tt.want.Monitors, got.Monitors
			if gm.CPU.ThresholdPercent != wm.CPU.ThresholdPercent {
				t.Errorf("CPU.ThresholdPercent = %v, want %v", gm.CPU.ThresholdPercent, wm.CPU.ThresholdPercent)
			}
			if gm.CPU.RecoveryPercent != wm.CPU.RecoveryPercent {
				t.Errorf("CPU.RecoveryPercent = %v, want %v", gm.CPU.RecoveryPercent, wm.CPU.RecoveryPercent)
			}
			if gm.Memory.ThresholdPercent != wm.Memory.ThresholdPercent {
				t.Errorf("Memory.ThresholdPercent = %v, want %v", gm.Memory.ThresholdPercent, wm.Memory.ThresholdPercent)
			}
			if gm.Memory.RecoveryPercent != wm.Memory.RecoveryPercent {
				t.Errorf("Memory.RecoveryPercent = %v, want %v", gm.Memory.RecoveryPercent, wm.Memory.RecoveryPercent)
			}
			if gm.Swap.ThresholdPercent != wm.Swap.ThresholdPercent {
				t.Errorf("Swap.ThresholdPercent = %v, want %v", gm.Swap.ThresholdPercent, wm.Swap.ThresholdPercent)
			}
			if gm.Swap.RecoveryPercent != wm.Swap.RecoveryPercent {
				t.Errorf("Swap.RecoveryPercent = %v, want %v", gm.Swap.RecoveryPercent, wm.Swap.RecoveryPercent)
			}
			if gm.Disk.ThresholdPercent != wm.Disk.ThresholdPercent {
				t.Errorf("Disk.ThresholdPercent = %v, want %v", gm.Disk.ThresholdPercent, wm.Disk.ThresholdPercent)
			}
			if gm.Disk.RecoveryPercent != wm.Disk.RecoveryPercent {
				t.Errorf("Disk.RecoveryPercent = %v, want %v", gm.Disk.RecoveryPercent, wm.Disk.RecoveryPercent)
			}
			if gm.Load.Load1.Warning != wm.Load.Load1.Warning {
				t.Errorf("Load1.Warning = %v, want %v", gm.Load.Load1.Warning, wm.Load.Load1.Warning)
			}
			if gm.Load.Load1.Critical != wm.Load.Load1.Critical {
				t.Errorf("Load1.Critical = %v, want %v", gm.Load.Load1.Critical, wm.Load.Load1.Critical)
			}
			if gm.Load.Load5.Warning != wm.Load.Load5.Warning {
				t.Errorf("Load5.Warning = %v, want %v", gm.Load.Load5.Warning, wm.Load.Load5.Warning)
			}
			if gm.Load.Load5.Critical != wm.Load.Load5.Critical {
				t.Errorf("Load5.Critical = %v, want %v", gm.Load.Load5.Critical, wm.Load.Load5.Critical)
			}
			if gm.Load.Load15.Warning != wm.Load.Load15.Warning {
				t.Errorf("Load15.Warning = %v, want %v", gm.Load.Load15.Warning, wm.Load.Load15.Warning)
			}
			if gm.Load.Load15.Critical != wm.Load.Load15.Critical {
				t.Errorf("Load15.Critical = %v, want %v", gm.Load.Load15.Critical, wm.Load.Load15.Critical)
			}
		})
	}
}

func Test_Load_WithMerging(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "config.yml")

	// Override GetConfigPath
	old := GetConfigPath
	GetConfigPath = func() (string, error) {
		return tmpFile, nil
	}
	defer func() { GetConfigPath = old }()

	tests := []struct {
		name             string
		userConfigYAML   string
		wantPollInterval int
		wantBotToken     string
		wantCPUThreshold float64
	}{
		{
			name:             "No user config file",
			userConfigYAML:   "",
			wantPollInterval: 15,
			wantBotToken:     "",
			wantCPUThreshold: 85.0,
		},
		{
			name:             "Empty user config file",
			userConfigYAML:   "",
			wantPollInterval: 15,
			wantBotToken:     "",
			wantCPUThreshold: 85.0,
		},
		{
			name:             "Partial user config — new nested format",
			userConfigYAML:   "poll_interval_seconds: 30\nbot_token: \"user_token\"",
			wantPollInterval: 30,
			wantBotToken:     "user_token",
			wantCPUThreshold: 85.0, // default preserved
		},
		{
			name: "Full user config — new nested format",
			userConfigYAML: `poll_interval_seconds: 60
bot_token: "full_user_token"
monitors:
  cpu:
    threshold_percent: 90.0`,
			wantPollInterval: 60,
			wantBotToken:     "full_user_token",
			wantCPUThreshold: 90.0,
		},
}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean up before each test
			os.Remove(tmpFile)

			// Write user config if provided
			if tt.userConfigYAML != "" {
				if err := os.WriteFile(tmpFile, []byte(tt.userConfigYAML), 0644); err != nil {
					t.Fatal(err)
				}
			}

			cfg, err := Load()
			if err != nil {
				t.Fatalf("Load() error = %v", err)
			}

			if cfg.PollInterval != tt.wantPollInterval {
				t.Errorf("PollInterval = %v, want %v", cfg.PollInterval, tt.wantPollInterval)
			}
			if cfg.BotToken != tt.wantBotToken {
				t.Errorf("BotToken = %v, want %v", cfg.BotToken, tt.wantBotToken)
			}
			if cfg.Monitors.CPU.ThresholdPercent != tt.wantCPUThreshold {
				t.Errorf("Monitors.CPU.ThresholdPercent = %v, want %v", cfg.Monitors.CPU.ThresholdPercent, tt.wantCPUThreshold)
			}
		})
	}
}


func Test_Load_WithInvalidUserConfig(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "config.yml")

	// Override GetConfigPath
	old := GetConfigPath
	GetConfigPath = func() (string, error) {
		return tmpFile, nil
	}
	defer func() { GetConfigPath = old }()

	// Write invalid user config
	invalidConfig := "poll_interval_seconds: -1\nbot_token: \"test\""
	if err := os.WriteFile(tmpFile, []byte(invalidConfig), 0644); err != nil {
		t.Fatal(err)
	}

	config, err := Load()
	if err != nil {
		t.Error("expected valid fallback config, got error:", err)
	}

	if config.PollInterval != 15 {
		t.Errorf("expected PollInterval = 15, got %v", config.PollInterval)
	}
}

func Test_UpdatePasswordHash(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "config.yml")

	// Override GetConfigPath
	old := GetConfigPath
	GetConfigPath = func() (string, error) {
		return tmpFile, nil
	}
	defer func() { GetConfigPath = old }()

	tests := []struct {
		name           string
		existingConfig string
		newHash        string
		expectedFields map[string]interface{}
	}{
		{
			name:           "No existing config file",
			existingConfig: "",
			newHash:        "argon2id$v=19$m=65536,t=3,p=4$test$test",
			expectedFields: map[string]interface{}{
				"join_password_hash": "argon2id$v=19$m=65536,t=3,p=4$test$test",
			},
		},
		{
			name:           "Existing config with other fields",
			existingConfig: "bot_token: \"user_token\"\npoll_interval_seconds: 30\nhostname_override: \"myhost\"",
			newHash:        "argon2id$v=19$m=65536,t=3,p=4$newhash$newhash",
			expectedFields: map[string]interface{}{
				"bot_token":             "user_token",
				"poll_interval_seconds": 30,
				"hostname_override":     "myhost",
				"join_password_hash":    "argon2id$v=19$m=65536,t=3,p=4$newhash$newhash",
			},
		},
		{
			name:           "Existing config with password hash",
			existingConfig: "bot_token: \"user_token\"\njoin_password_hash: \"old_hash\"",
			newHash:        "argon2id$v=19$m=65536,t=3,p=4$updated$updated",
			expectedFields: map[string]interface{}{
				"bot_token":          "user_token",
				"join_password_hash": "argon2id$v=19$m=65536,t=3,p=4$updated$updated",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean up before each test
			os.Remove(tmpFile)

			// Write existing config if provided
			if tt.existingConfig != "" {
				if err := os.WriteFile(tmpFile, []byte(tt.existingConfig), 0644); err != nil {
					t.Fatal(err)
				}
			}

			// Update password hash
			if err := UpdatePasswordHash(tt.newHash); err != nil {
				t.Fatalf("UpdatePasswordHash() error = %v", err)
			}

			// Read the resulting config and parse it
			data, err := os.ReadFile(tmpFile)
			if err != nil {
				t.Fatalf("Error reading config file: %v", err)
			}

			var result map[string]interface{}
			if err := yaml.Unmarshal(data, &result); err != nil {
				t.Fatalf("Error parsing resulting config: %v", err)
			}

			// Check that all expected fields are present with correct values
			for key, expectedValue := range tt.expectedFields {
				if actualValue, exists := result[key]; !exists {
					t.Errorf("Expected field '%s' not found in config", key)
				} else if actualValue != expectedValue {
					t.Errorf("Field '%s' value mismatch. Got: %v, Expected: %v", key, actualValue, expectedValue)
				}
			}

			// Check that no unexpected fields are present (only for the first test case)
			if tt.name == "No existing config file" {
				if len(result) != 1 {
					t.Errorf("Expected exactly 1 field in config, got %d: %v", len(result), result)
				}
			}
		})
	}
}

func Test_expandPath(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("Failed to get user home directory: %v", err)
	}

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Path with ~",
			input:    "~/.config/tg-system-monitor/tgsm.db",
			expected: filepath.Join(home, ".config/tg-system-monitor/tgsm.db"),
		},
		{
			name:     "Path with ~ and simple file",
			input:    "~/test.db",
			expected: filepath.Join(home, "test.db"),
		},
		{
			name:     "Absolute path without ~",
			input:    "/tmp/test.db",
			expected: "/tmp/test.db",
		},
		{
			name:     "Relative path without ~",
			input:    "./local.db",
			expected: "./local.db",
		},
		{
			name:     "Empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "Just ~",
			input:    "~",
			expected: home,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := expandPath(tt.input)
			if result != tt.expected {
				t.Errorf("expandPath(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func Test_Load_DBPathExpansion(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "config.yml")

	// Override GetConfigPath
	old := GetConfigPath
	GetConfigPath = func() (string, error) {
		return tmpFile, nil
	}
	defer func() { GetConfigPath = old }()

	// Write user config with ~ in db_path
	userConfig := `db_path: "~/test_expansion.db"`
	if err := os.WriteFile(tmpFile, []byte(userConfig), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("Failed to get home directory: %v", err)
	}

	expectedPath := filepath.Join(home, "test_expansion.db")
	if cfg.DBPath != expectedPath {
		t.Errorf("DBPath expansion failed. Expected: %s, Got: %s", expectedPath, cfg.DBPath)
	}
}
