package message

import "fmt"

// Standard status terms
const (
	StatusStarted    = "started"
	StatusCompleted  = "completed"
	StatusFailed     = "failed"
	StatusTriggered  = "triggered"
	StatusRecovered  = "recovered"
	StatusDisabled   = "disabled"
	StatusEnabled    = "enabled"
	StatusIgnored    = "ignored"
	StatusDeprecated = "deprecated"
)

// Log message templates
var (
	LogStarted = func(component, service string) string {
		return fmt.Sprintf("[%s] %s: %s", component, StatusStarted, service)
	}
	LogCompleted = func(component, operation string) string {
		return fmt.Sprintf("[%s] %s: %s", component, StatusCompleted, operation)
	}
	LogFailed = func(component, operation, error string) string {
		return fmt.Sprintf("[%s] %s: %s - %s", component, StatusFailed, operation, error)
	}
	LogTriggered = func(component, alertType, value string) string {
		return fmt.Sprintf("[%s] %s: %s - %s", component, StatusTriggered, alertType, value)
	}
	LogRecovered = func(component, alertType, value string) string {
		return fmt.Sprintf("[%s] %s: %s - %s", component, StatusRecovered, alertType, value)
	}
	LogDisabled = func(component, feature string) string {
		return fmt.Sprintf("[%s] %s: %s", component, StatusDisabled, feature)
	}
	LogEnabled = func(component, feature string) string {
		return fmt.Sprintf("[%s] %s: %s", component, StatusEnabled, feature)
	}
	LogIgnored = func(component, condition, reason string) string {
		return fmt.Sprintf("[%s] %s: %s - %s", component, StatusIgnored, condition, reason)
	}
	LogDeprecated = func(component, oldKey, newKey string) string {
		return fmt.Sprintf("[%s] %s: %q is deprecated, use %q instead", component, StatusDeprecated, oldKey, newKey)
	}
)

// User response templates
var (
	SuccessTemplate = func(action, details string) string {
		return fmt.Sprintf("✅ *%s*\n\n**Details:** %s", action, details)
	}

	ErrorTemplate = func(operation, error, action string) string {
		return fmt.Sprintf("❌ *%s*\n\n**Error:** %s\n**Action:** %s", operation, error, action)
	}

	StatusTemplate = func(component, status, info string) string {
		return fmt.Sprintf("🔍 *Status Report*\n\n**Component:** %s\n**Status:** %s\n**Details:** %s", component, status, info)
	}

	AlertTemplate = func(alertType, severity, metric, value, transition string) string {
		if transition == "recovered" {
			return fmt.Sprintf("✅ *%s RECOVERED*\n\n**Metric:** %s\n**Value:** %s", alertType, metric, value)
		}

		emoji := "⚠️"
		if severity == "critical" {
			emoji = "🚨"
		}
		return fmt.Sprintf("%s *%s %s*\n\n**Severity:** %s\n**Metric:** %s\n**Value:** %s", emoji, alertType, severity, severity, metric, value)
	}

	AuthDeniedTemplate = func(reason string) string {
		return fmt.Sprintf("🚫 *Access Denied*\n\n%s", reason)
	}

	AuthSuccessTemplate = func(user, username string, id int64) string {
		return fmt.Sprintf("✅ *Authentication Successful*\n\n👤 **User:** %s (@%s)\n🆔 **ID:** `%d`\n🔐 **Status:** Authorized to use restricted commands", user, username, id)
	}
)

// Common message constants
const (
	ConfigError         = "Configuration loading failed"
	DatabaseError       = "Database operation failed"
	AuthenticationError = "Authentication failed"
	TelegramError       = "Telegram bot operation failed"
	MetricsError        = "Metrics collection failed"
	AlertError          = "Alert processing failed"

	BotTokenRequired = "Bot token is required in config"
	PasswordRequired = "Password required for restricted commands"
	InvalidPassword  = "Invalid password"
	PasswordEmpty    = "Password cannot be empty"
	PasswordMismatch = "Passwords do not match"

	CommandRestricted = "Command is restricted"
	CommandNotFound   = "Command not found"
	InvalidArguments  = "Invalid arguments provided"

	DatabaseHealthy   = "Database connection healthy"
	DatabaseUnhealthy = "Database connection failed"

	NoMetricsAvailable = "No system metrics available yet"
	UserNotFound       = "User not found in system"
	UserRemoved        = "User removed from system"
	AlertsEnabled      = "Alert notifications enabled"
	AlertsDisabled     = "Alert notifications disabled"
)

// Component names for consistent logging
const (
	ComponentMain      = "main"
	ComponentBot       = "bot"
	ComponentAuth      = "auth"
	ComponentDatabase  = "database"
	ComponentDetection = "detection"
	ComponentAlert     = "alert"
	ComponentMetrics   = "metrics"
	ComponentConfig    = "config"
	ComponentAPI       = "api"
)
