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
