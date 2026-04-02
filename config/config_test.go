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

func Test_mergeConfigs(t *testing.T) {
	defaultConfig := &Config{
		BotToken:             "default_token",
		JoinPasswordHash:     "default_hash",
		HostnameOverride:     "default_host",
		PollInterval:         15,
		CPUThresholdPercent:  85.0,
		CPURecoveryPercent:   70.0,
		CPUSustainSeconds:    300,
		MemThresholdPercent:  90.0,
		MemRecoveryPercent:   80.0,
		MemSustainSeconds:    180,
		SwapThresholdPercent: 25.0,
		SwapRecoveryPercent:  10.0,
		SwapSustainSeconds:   180,
		DiskThresholdPercent: 95.0,
		DiskRecoveryPercent:  90.0,
		Load1Warning:         2.0,
		Load1Critical:        4.0,
		Load5Warning:         1.5,
		Load5Critical:        3.0,
		Load15Warning:        1.0,
		Load15Critical:       2.0,
		Hysteresis:           5.0,
		AlertCooldownSeconds: 1800,
		TopProcessCount:      5,
		DBPath:               "/default/path",
	}

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
			name: "Partial override",
			userConfig: &Config{
				BotToken:     "user_token",
				PollInterval: 30,
			},
			want: &Config{
				BotToken:             "user_token",    // overridden
				JoinPasswordHash:     "default_hash",  // preserved
				HostnameOverride:     "default_host",  // preserved
				PollInterval:         30,              // overridden
				CPUThresholdPercent:  85.0,            // preserved
				CPURecoveryPercent:   70.0,            // preserved
				CPUSustainSeconds:    300,             // preserved
				MemThresholdPercent:  90.0,            // preserved
				MemRecoveryPercent:   80.0,            // preserved
				MemSustainSeconds:    180,             // preserved
				SwapThresholdPercent: 25.0,            // preserved
				SwapRecoveryPercent:  10.0,            // preserved
				SwapSustainSeconds:   180,             // preserved
				DiskThresholdPercent: 95.0,            // preserved
				DiskRecoveryPercent:  90.0,            // preserved
				Load1Warning:         2.0,             // preserved
				Load1Critical:        4.0,             // preserved
				Load5Warning:         1.5,             // preserved
				Load5Critical:        3.0,             // preserved
				Load15Warning:        1.0,             // preserved
				Load15Critical:       2.0,             // preserved
				Hysteresis:           5.0,             // preserved
				AlertCooldownSeconds: 1800,            // preserved
				TopProcessCount:      5,               // preserved
				DBPath:               "/default/path", // preserved
			},
		},
		{
			name: "Full override",
			userConfig: &Config{
				BotToken:             "user_token",
				JoinPasswordHash:     "user_hash",
				HostnameOverride:     "user_host",
				PollInterval:         60,
				CPUThresholdPercent:  90.0,
				CPURecoveryPercent:   75.0,
				CPUSustainSeconds:    600,
				MemThresholdPercent:  95.0,
				MemRecoveryPercent:   85.0,
				MemSustainSeconds:    300,
				SwapThresholdPercent: 30.0,
				SwapRecoveryPercent:  15.0,
				SwapSustainSeconds:   300,
				DiskThresholdPercent: 98.0,
				DiskRecoveryPercent:  95.0,
				Load1Warning:         3.0,
				Load1Critical:        5.0,
				Load5Warning:         2.0,
				Load5Critical:        4.0,
				Load15Warning:        1.5,
				Load15Critical:       2.5,
				Hysteresis:           10.0,
				AlertCooldownSeconds: 3600,
				TopProcessCount:      10,
				DBPath:               "/user/path",
			},
			want: &Config{
				BotToken:             "user_token",
				JoinPasswordHash:     "user_hash",
				HostnameOverride:     "user_host",
				PollInterval:         60,
				CPUThresholdPercent:  90.0,
				CPURecoveryPercent:   75.0,
				CPUSustainSeconds:    600,
				MemThresholdPercent:  95.0,
				MemRecoveryPercent:   85.0,
				MemSustainSeconds:    300,
				SwapThresholdPercent: 30.0,
				SwapRecoveryPercent:  15.0,
				SwapSustainSeconds:   300,
				DiskThresholdPercent: 98.0,
				DiskRecoveryPercent:  95.0,
				Load1Warning:         3.0,
				Load1Critical:        5.0,
				Load5Warning:         2.0,
				Load5Critical:        4.0,
				Load15Warning:        1.5,
				Load15Critical:       2.5,
				Hysteresis:           10.0,
				AlertCooldownSeconds: 3600,
				TopProcessCount:      10,
				DBPath:               "/user/path",
			},
		},
		{
			name: "Undefined poll_interval_seconds in user config",
			userConfig: &Config{
				BotToken: "user_token",
				// PollInterval is undefined (zero value)
			},
			want: &Config{
				BotToken:             "user_token",    // overridden
				JoinPasswordHash:     "default_hash",  // preserved
				HostnameOverride:     "default_host",  // preserved
				PollInterval:         15,              // should use default value
				CPUThresholdPercent:  85.0,            // preserved
				CPURecoveryPercent:   70.0,            // preserved
				CPUSustainSeconds:    300,             // preserved
				MemThresholdPercent:  90.0,            // preserved
				MemRecoveryPercent:   80.0,            // preserved
				MemSustainSeconds:    180,             // preserved
				SwapThresholdPercent: 25.0,            // preserved
				SwapRecoveryPercent:  10.0,            // preserved
				SwapSustainSeconds:   180,             // preserved
				DiskThresholdPercent: 95.0,            // preserved
				DiskRecoveryPercent:  90.0,            // preserved
				Load1Warning:         2.0,             // preserved
				Load1Critical:        4.0,             // preserved
				Load5Warning:         1.5,             // preserved
				Load5Critical:        3.0,             // preserved
				Load15Warning:        1.0,             // preserved
				Load15Critical:       2.0,             // preserved
				Hysteresis:           5.0,             // preserved
				AlertCooldownSeconds: 1800,            // preserved
				TopProcessCount:      5,               // preserved
				DBPath:               "/default/path", // preserved
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
			if got.CPUThresholdPercent != tt.want.CPUThresholdPercent {
				t.Errorf("CPUThresholdPercent = %v, want %v", got.CPUThresholdPercent, tt.want.CPUThresholdPercent)
			}
			if got.CPURecoveryPercent != tt.want.CPURecoveryPercent {
				t.Errorf("CPURecoveryPercent = %v, want %v", got.CPURecoveryPercent, tt.want.CPURecoveryPercent)
			}
			if got.CPUSustainSeconds != tt.want.CPUSustainSeconds {
				t.Errorf("CPUSustainSeconds = %v, want %v", got.CPUSustainSeconds, tt.want.CPUSustainSeconds)
			}
			if got.MemThresholdPercent != tt.want.MemThresholdPercent {
				t.Errorf("MemThresholdPercent = %v, want %v", got.MemThresholdPercent, tt.want.MemThresholdPercent)
			}
			if got.MemRecoveryPercent != tt.want.MemRecoveryPercent {
				t.Errorf("MemRecoveryPercent = %v, want %v", got.MemRecoveryPercent, tt.want.MemRecoveryPercent)
			}
			if got.MemSustainSeconds != tt.want.MemSustainSeconds {
				t.Errorf("MemSustainSeconds = %v, want %v", got.MemSustainSeconds, tt.want.MemSustainSeconds)
			}
			if got.SwapThresholdPercent != tt.want.SwapThresholdPercent {
				t.Errorf("SwapThresholdPercent = %v, want %v", got.SwapThresholdPercent, tt.want.SwapThresholdPercent)
			}
			if got.SwapRecoveryPercent != tt.want.SwapRecoveryPercent {
				t.Errorf("SwapRecoveryPercent = %v, want %v", got.SwapRecoveryPercent, tt.want.SwapRecoveryPercent)
			}
			if got.SwapSustainSeconds != tt.want.SwapSustainSeconds {
				t.Errorf("SwapSustainSeconds = %v, want %v", got.SwapSustainSeconds, tt.want.SwapSustainSeconds)
			}
			if got.DiskThresholdPercent != tt.want.DiskThresholdPercent {
				t.Errorf("DiskThresholdPercent = %v, want %v", got.DiskThresholdPercent, tt.want.DiskThresholdPercent)
			}
			if got.DiskRecoveryPercent != tt.want.DiskRecoveryPercent {
				t.Errorf("DiskRecoveryPercent = %v, want %v", got.DiskRecoveryPercent, tt.want.DiskRecoveryPercent)
			}
			if got.Load1Warning != tt.want.Load1Warning {
				t.Errorf("Load1Warning = %v, want %v", got.Load1Warning, tt.want.Load1Warning)
			}
			if got.Load1Critical != tt.want.Load1Critical {
				t.Errorf("Load1Critical = %v, want %v", got.Load1Critical, tt.want.Load1Critical)
			}
			if got.Load5Warning != tt.want.Load5Warning {
				t.Errorf("Load5Warning = %v, want %v", got.Load5Warning, tt.want.Load5Warning)
			}
			if got.Load5Critical != tt.want.Load5Critical {
				t.Errorf("Load5Critical = %v, want %v", got.Load5Critical, tt.want.Load5Critical)
			}
			if got.Load15Warning != tt.want.Load15Warning {
				t.Errorf("Load15Warning = %v, want %v", got.Load15Warning, tt.want.Load15Warning)
			}
			if got.Load15Critical != tt.want.Load15Critical {
				t.Errorf("Load15Critical = %v, want %v", got.Load15Critical, tt.want.Load15Critical)
			}
			if got.Hysteresis != tt.want.Hysteresis {
				t.Errorf("Hysteresis = %v, want %v", got.Hysteresis, tt.want.Hysteresis)
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
			wantPollInterval: 15,                    // default
			wantBotToken:     "YOUR_BOT_TOKEN_HERE", // default from default-config.yml
			wantCPUThreshold: 85.0,                  // default
		},
		{
			name:             "Empty user config file",
			userConfigYAML:   "",
			wantPollInterval: 15,                    // default
			wantBotToken:     "YOUR_BOT_TOKEN_HERE", // default from default-config.yml
			wantCPUThreshold: 85.0,                  // default
		},
		{
			name:             "Partial user config",
			userConfigYAML:   "poll_interval_seconds: 30\nbot_token: \"user_token\"",
			wantPollInterval: 30,           // overridden
			wantBotToken:     "user_token", // overridden
			wantCPUThreshold: 85.0,         // preserved default
		},
		{
			name:             "Full user config",
			userConfigYAML:   "poll_interval_seconds: 60\nbot_token: \"full_user_token\"\ncpu_threshold_percent: 90.0",
			wantPollInterval: 60,                // overridden
			wantBotToken:     "full_user_token", // overridden
			wantCPUThreshold: 90.0,              // overridden
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
			if cfg.CPUThresholdPercent != tt.wantCPUThreshold {
				t.Errorf("CPUThresholdPercent = %v, want %v", cfg.CPUThresholdPercent, tt.wantCPUThreshold)
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
