package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"log"
	"strings"
	"tg-system-monitor/db"
	"time"

	"golang.org/x/crypto/argon2"
)

const (
	// Argon2 parameters
	argonTime    = 3
	argonMemory  = 64 * 1024
	argonThreads = 4
	argonKeyLen  = 32
	argonSaltLen = 16

	// Failed authentication limits
	maxFailedAttempts   = 5
	failedAttemptWindow = 15 * time.Minute
)

// Restricted commands that require authentication
var restrictedCommands = map[string]bool{
	"join":     true,
	"leave":    true,
	"allow":    true,
	"disallow": true,
	"alerts":   true,
	"config":   true,
	"shutdown": true,
}

type AuthManager struct {
	db           *db.DB
	passwordHash string
	allowedUsers map[int64]bool
}

type AuthResult struct {
	Authorized bool
	Reason     string
}

func NewAuthManager(database *db.DB, passwordHash string) *AuthManager {
	return &AuthManager{
		db:           database,
		passwordHash: passwordHash,
		allowedUsers: make(map[int64]bool),
	}
}

// HashPassword generates an Argon2 hash for the given password
func HashPassword(password string) (string, error) {
	salt := make([]byte, argonSaltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("failed to generate salt: %w", err)
	}

	hash := argon2.IDKey([]byte(password), salt, argonTime, argonMemory, argonThreads, argonKeyLen)

	// Format: $argon2id$v=19$m=65536,t=3,p=4$base64(salt)$base64(hash)
	saltB64 := base64.RawStdEncoding.EncodeToString(salt)
	hashB64 := base64.RawStdEncoding.EncodeToString(hash)

	return fmt.Sprintf("$argon2id$v=19$m=%d,t=%d,p=%d$%s$%s",
		argonMemory, argonTime, argonThreads, saltB64, hashB64), nil
}

// VerifyPassword verifies a password against an Argon2 hash
func VerifyPassword(password, hash string) bool {
	// Parse the hash format using manual parsing since sscanf has issues with base64
	parts := strings.Split(hash, "$")
	if len(parts) != 6 || parts[0] != "" || parts[1] != "argon2id" || parts[2] != "v=19" {
		log.Printf("Invalid hash format")
		return false
	}

	// Parse parameters
	paramParts := strings.Split(parts[3], ",")
	if len(paramParts) != 3 {
		log.Printf("Invalid parameter format")
		return false
	}

	var memory uint32
	var time uint32
	var threads uint8

	for _, param := range paramParts {
		if strings.HasPrefix(param, "m=") {
			_, err := fmt.Sscanf(param, "m=%d", &memory)
			if err != nil {
				log.Printf("Failed to parse memory parameter: %v", err)
				return false
			}
		} else if strings.HasPrefix(param, "t=") {
			_, err := fmt.Sscanf(param, "t=%d", &time)
			if err != nil {
				log.Printf("Failed to parse time parameter: %v", err)
				return false
			}
		} else if strings.HasPrefix(param, "p=") {
			_, err := fmt.Sscanf(param, "p=%d", &threads)
			if err != nil {
				log.Printf("Failed to parse threads parameter: %v", err)
				return false
			}
		}
	}

	// Decode salt and hash
	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		log.Printf("Failed to decode salt: %v", err)
		return false
	}

	hashBytes, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		log.Printf("Failed to decode hash: %v", err)
		return false
	}

	hashLen := len(hashBytes)

	// Generate hash with the same parameters
	computedHash := argon2.IDKey([]byte(password), salt, time, memory, threads, uint32(hashLen))

	// Constant-time comparison
	return subtle.ConstantTimeCompare(hashBytes, computedHash) == 1
}

// IsRestrictedCommand checks if a command requires authentication
func (am *AuthManager) IsRestrictedCommand(command string) bool {
	return restrictedCommands[command]
}

// AuthenticateUser verifies if a user is authorized to execute a command
func (am *AuthManager) AuthenticateUser(userID int64, username string, command string, password string) AuthResult {
	// Check if command is restricted
	if !am.IsRestrictedCommand(command) {
		return AuthResult{Authorized: true, Reason: "Command is not restricted"}
	}

	// Check failed authentication attempts
	failCount, lastFail, err := am.db.GetFailedAuth(userID)
	if err != nil {
		log.Printf("Failed to get auth attempts for user %d: %v", userID, err)
		return AuthResult{Authorized: false, Reason: "Authentication system error"}
	}

	// Rate limit failed attempts
	if failCount >= maxFailedAttempts {
		timeSinceLastFail := time.Since(time.Unix(lastFail, 0))
		if timeSinceLastFail < failedAttemptWindow {
			remainingTime := failedAttemptWindow - timeSinceLastFail
			return AuthResult{
				Authorized: false,
				Reason:     fmt.Sprintf("Too many failed attempts. Try again in %v", remainingTime.Round(time.Minute)),
			}
		}
		// Reset failed attempts if window has passed
		if err := am.db.ResetFailedAuth(userID); err != nil {
			log.Printf("Failed to reset auth attempts for user %d: %v", userID, err)
		}
	}

	// Verify password if provided
	if password == "" {
		if err := am.db.IncrementFailedAuth(userID); err != nil {
			log.Printf("Failed to increment auth attempts for user %d: %v", userID, err)
		}
		return AuthResult{Authorized: false, Reason: "Password required for restricted commands"}
	}

	if !VerifyPassword(password, am.passwordHash) {
		if err := am.db.IncrementFailedAuth(userID); err != nil {
			log.Printf("Failed to increment auth attempts for user %d: %v", userID, err)
		}
		return AuthResult{Authorized: false, Reason: "Invalid password"}
	}

	// Check if user is in allowlist
	user, err := am.db.GetUser(userID)
	if err != nil {
		log.Printf("Failed to get user %d from database: %v", userID, err)
		return AuthResult{Authorized: false, Reason: "Database error"}
	}

	if user == nil || !user.IsAllowed {
		if err := am.db.IncrementFailedAuth(userID); err != nil {
			log.Printf("Failed to increment auth attempts for user %d: %v", userID, err)
		}
		return AuthResult{Authorized: false, Reason: "User not in allowlist"}
	}

	// Reset failed attempts on successful authentication
	if err := am.db.ResetFailedAuth(userID); err != nil {
		log.Printf("Failed to reset auth attempts for user %d: %v", userID, err)
	}

	return AuthResult{Authorized: true, Reason: "Authentication successful"}
}

// AddUserToAllowlist adds a user to the allowlist
func (am *AuthManager) AddUserToAllowlist(userID int64, username, firstName, lastName string) error {
	user := &db.User{
		ID:            userID,
		Username:      username,
		FirstName:     firstName,
		LastName:      lastName,
		JoinedAt:      time.Now(),
		IsAllowed:     true,
		LastSeenAt:    time.Now(),
		AlertsEnabled: true, // Enable alerts by default
	}

	return am.db.UpdateUser(user)
}

// RemoveUserFromAllowlist removes a user from the allowlist
func (am *AuthManager) RemoveUserFromAllowlist(userID int64) error {
	user, err := am.db.GetUser(userID)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}
	if user == nil {
		return fmt.Errorf("user not found")
	}

	user.IsAllowed = false
	user.LastSeenAt = time.Now()
	return am.db.UpdateUser(user)
}

// IsUserAllowed checks if a user is in the allowlist
func (am *AuthManager) IsUserAllowed(userID int64) (bool, error) {
	user, err := am.db.GetUser(userID)
	if err != nil {
		return false, err
	}
	return user != nil && user.IsAllowed, nil
}

// GetDatabase returns the database instance for the auth manager
func (am *AuthManager) GetDatabase() *db.DB {
	return am.db
}

// GetRestrictedCommands returns a list of all restricted commands
func GetRestrictedCommands() []string {
	commands := make([]string, 0, len(restrictedCommands))
	for cmd := range restrictedCommands {
		commands = append(commands, cmd)
	}
	return commands
}
