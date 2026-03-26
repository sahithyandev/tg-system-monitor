package auth

import (
	"testing"
)

func TestHashPassword(t *testing.T) {
	password := "testpassword123"
	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("Failed to hash password: %v", err)
	}

	if hash == "" {
		t.Fatal("Hash is empty")
	}

	// Debug: print the hash format
	t.Logf("Generated hash: %s", hash)

	// Verify hash format
	if len(hash) < 10 {
		t.Fatal("Hash is too short")
	}
}

func TestVerifyPassword(t *testing.T) {
	password := "testpassword123"
	wrongPassword := "wrongpassword"

	// Hash password
	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("Failed to hash password: %v", err)
	}

	// Test correct password
	if !VerifyPassword(password, hash) {
		t.Error("Correct password verification failed")
	}

	// Test wrong password
	if VerifyPassword(wrongPassword, hash) {
		t.Error("Wrong password verification succeeded")
	}

	// Test invalid hash format
	if VerifyPassword(password, "invalid_hash") {
		t.Error("Invalid hash verification succeeded")
	}
}

func TestIsRestrictedCommand(t *testing.T) {
	authManager := &AuthManager{}

	// Test restricted commands
	restricted := []string{"join", "leave", "allow", "disallow", "alerts", "config", "shutdown"}
	for _, cmd := range restricted {
		if !authManager.IsRestrictedCommand(cmd) {
			t.Errorf("Command %s should be restricted", cmd)
		}
	}

	// Test non-restricted commands
	nonRestricted := []string{"ping", "whoami", "help", "start"}
	for _, cmd := range nonRestricted {
		if authManager.IsRestrictedCommand(cmd) {
			t.Errorf("Command %s should not be restricted", cmd)
		}
	}
}

func TestGetRestrictedCommands(t *testing.T) {
	commands := GetRestrictedCommands()

	if len(commands) == 0 {
		t.Fatal("No restricted commands returned")
	}

	// Check that all expected commands are present
	expected := map[string]bool{
		"join":     true,
		"leave":    true,
		"allow":    true,
		"disallow": true,
		"alerts":   true,
		"config":   true,
		"shutdown": true,
	}

	for _, cmd := range commands {
		if !expected[cmd] {
			t.Errorf("Unexpected restricted command: %s", cmd)
		}
		delete(expected, cmd)
	}

	if len(expected) > 0 {
		t.Errorf("Missing restricted commands: %v", expected)
	}
}

func TestAuthResult(t *testing.T) {
	// Test successful auth
	result := AuthResult{Authorized: true, Reason: "Success"}
	if !result.Authorized {
		t.Error("Expected authorized to be true")
	}
	if result.Reason != "Success" {
		t.Errorf("Expected reason to be 'Success', got '%s'", result.Reason)
	}

	// Test failed auth
	result = AuthResult{Authorized: false, Reason: "Invalid password"}
	if result.Authorized {
		t.Error("Expected authorized to be false")
	}
	if result.Reason != "Invalid password" {
		t.Errorf("Expected reason to be 'Invalid password', got '%s'", result.Reason)
	}
}

func TestPasswordHashConsistency(t *testing.T) {
	password := "testpassword123"

	// Hash password multiple times
	hash1, err1 := HashPassword(password)
	hash2, err2 := HashPassword(password)

	if err1 != nil || err2 != nil {
		t.Fatalf("Failed to hash password: %v, %v", err1, err2)
	}

	// Hashes should be different (due to random salt)
	if hash1 == hash2 {
		t.Error("Hashes should be different due to random salt")
	}

	// But both should verify the password
	if !VerifyPassword(password, hash1) {
		t.Error("First hash verification failed")
	}
	if !VerifyPassword(password, hash2) {
		t.Error("Second hash verification failed")
	}
}
