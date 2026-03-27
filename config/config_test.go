package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     Config
		wantErr bool
	}{
		{"Valid interval", Config{PollInterval: 15}, false},
		{"Minimum valid", Config{PollInterval: 1}, false},
		{"Zero interval", Config{PollInterval: 0}, true},
		{"Negative interval", Config{PollInterval: -1}, true},
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
	_, err = Load()
	if err == nil {
		t.Error("expected error for invalid config, got nil")
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
				AlertCooldownSeconds: 3600,
				TopProcessCount:      10,
				DBPath:               "/user/path",
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
			wantPollInterval: 15,   // default
			wantBotToken:     "",   // default from default-config.yml
			wantCPUThreshold: 85.0, // default
		},
		{
			name:             "Empty user config file",
			userConfigYAML:   "",
			wantPollInterval: 15,   // default
			wantBotToken:     "",   // default from default-config.yml
			wantCPUThreshold: 85.0, // default
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

	_, err := Load()
	if err == nil {
		t.Error("expected error for invalid user config, got nil")
	}
}
