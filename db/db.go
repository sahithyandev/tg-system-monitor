package db

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"math/rand/v2"
	"os"
	"path/filepath"
	"time"

	msg "tg-system-monitor/message"
	"tg-system-monitor/metrics"

	_ "modernc.org/sqlite"
)

type User struct {
	ID            int64
	Username      string
	FirstName     string
	LastName      string
	FirstAuthAt   time.Time
	LastAuthAt    time.Time
	AlertsEnabled bool
	CreatedAt     time.Time
}

type AlertState struct {
	Key               string
	IsActive          bool
	ActiveSinceUnix   int64
	LastTriggeredUnix int64
	LastRecoveredUnix int64
}

type DB struct {
	conn *sql.DB
}

func Init(path string) (*DB, error) {
	// Ensure the database directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %w", err)
	}

	conn, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Configure connection pool for better concurrency
	conn.SetMaxOpenConns(10)                  // More conservative connection limit
	conn.SetMaxIdleConns(3)                   // Keep fewer idle connections
	conn.SetConnMaxLifetime(10 * time.Minute) // Longer connection lifetime

	db := &DB{conn: conn}

	// Configure SQLite for better concurrency
	if err := db.configureSQLite(); err != nil {
		return nil, fmt.Errorf("failed to configure sqlite: %w", err)
	}

	if err := db.Migrate(); err != nil {
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}

	return db, nil
}

func (db *DB) Close() error {
	return db.conn.Close()
}

// configureSQLite sets up SQLite for better concurrency
func (db *DB) configureSQLite() error {
	configs := []string{
		// Enable WAL mode for better concurrent read/write performance
		"PRAGMA journal_mode=WAL;",
		// Set busy timeout to 60 seconds to handle temporary locks
		"PRAGMA busy_timeout=60000;",
		// Enable foreign key constraints
		"PRAGMA foreign_keys=ON;",
		// Optimize for better performance
		"PRAGMA synchronous=NORMAL;",
		"PRAGMA cache_size=10000;",
		"PRAGMA temp_store=MEMORY;",
		// Additional concurrency improvements
		"PRAGMA wal_autocheckpoint=1000;",
		"PRAGMA locking_mode=NORMAL;",
	}

	for _, config := range configs {
		if _, err := db.conn.Exec(config); err != nil {
			return fmt.Errorf("failed to execute %q: %w", config, err)
		}
	}

	return nil
}

func (db *DB) GetConn() *sql.DB {
	return db.conn
}

func (db *DB) Ping() error {
	return db.conn.Ping()
}

// executeWithRetry executes a database operation with retry logic for transient errors
func (db *DB) executeWithRetry(ctx context.Context, fn func() error) error {
	const maxRetries = 10
	const baseDelay = 50 * time.Millisecond

	var lastErr error
	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			// Exponential backoff with jitter
			delay := baseDelay * time.Duration(1<<uint(attempt-1))
			if delay > 5*time.Second {
				delay = 5 * time.Second
			}
			// Add jitter to avoid thundering herd
			jitter := time.Duration(float64(delay) * 0.2 * (2.0*rand.Float64() - 1.0))
			delay += jitter

			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
			}
		}

		if err := fn(); err != nil {
			lastErr = err

			// Check if this is a retryable error
			if isRetryableError(err) {
				continue // Retry
			}

			return err // Non-retryable error, return immediately
		}

		return nil // Success
	}

	return fmt.Errorf("operation failed after %d attempts, last error: %w", maxRetries, lastErr)
}

// isRetryableError checks if an error is worth retrying
func isRetryableError(err error) bool {
	if err == nil {
		return false
	}

	errStr := err.Error()
	// SQLite busy errors
	if contains(errStr, "database is locked") || contains(errStr, "SQLITE_BUSY") {
		return true
	}

	// Connection errors
	if contains(errStr, "connection") && contains(errStr, "refused") {
		return true
	}

	return false
}

// contains checks if a string contains a substring (case-insensitive)
func contains(s, substr string) bool {
	return len(s) >= len(substr) &&
		(s == substr ||
			len(s) > len(substr) &&
				(s[:len(substr)] == substr ||
					s[len(s)-len(substr):] == substr ||
					containsMiddle(s, substr)))
}

func containsMiddle(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func (db *DB) Migrate() error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS users (
			telegram_user_id INTEGER PRIMARY KEY,
			username TEXT,
			first_name TEXT,
			last_name TEXT,
			first_auth_at TEXT,
			last_auth_at TEXT,
			alerts_enabled INTEGER NOT NULL DEFAULT 1,
			created_at TEXT DEFAULT CURRENT_TIMESTAMP
		);`,
		`CREATE TABLE IF NOT EXISTS settings (
			key TEXT PRIMARY KEY,
			value TEXT NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS alert_state (
			alert_key TEXT PRIMARY KEY,
			is_active INTEGER NOT NULL DEFAULT 0,
			active_since_unix INTEGER,
			last_triggered_unix INTEGER,
			last_recovered_unix INTEGER
		);`,
		`CREATE TABLE IF NOT EXISTS failed_auth (
			telegram_user_id INTEGER PRIMARY KEY,
			fail_count INTEGER NOT NULL DEFAULT 0,
			last_fail_unix INTEGER
		);`,
		`CREATE TABLE IF NOT EXISTS metric_samples (
			ts_unix INTEGER NOT NULL,
			cpu_percent REAL,
			mem_percent REAL,
			swap_percent REAL,
			disk_percent REAL,
			load1 REAL,
			load5 REAL,
			load15 REAL,
			uptime REAL
		);`,
		`CREATE TABLE IF NOT EXISTS volume_samples (
			ts_unix INTEGER NOT NULL,
			mount_path TEXT NOT NULL,
			disk_percent REAL
		);`,
		`CREATE INDEX IF NOT EXISTS idx_volume_samples_path_ts
			ON volume_samples(mount_path, ts_unix DESC);`,
		`CREATE TABLE IF NOT EXISTS alerts (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			alert_type TEXT NOT NULL,
			severity TEXT NOT NULL,
			value REAL NOT NULL,
			threshold REAL NOT NULL,
			message TEXT NOT NULL,
			transition TEXT NOT NULL,
			timestamp_unix INTEGER NOT NULL,
			sent_at_unix INTEGER,
			created_at TEXT DEFAULT CURRENT_TIMESTAMP
		);`,
		`CREATE INDEX IF NOT EXISTS idx_alerts_unsent ON alerts(sent_at_unix) WHERE sent_at_unix IS NULL;`,
		`CREATE INDEX IF NOT EXISTS idx_alerts_timestamp ON alerts(timestamp_unix DESC);`,
		`CREATE INDEX IF NOT EXISTS idx_alert_state_active ON alert_state(is_active);`,
		`CREATE INDEX IF NOT EXISTS idx_alert_state_last_triggered ON alert_state(last_triggered_unix);`,
		`CREATE INDEX IF NOT EXISTS idx_metric_samples_ts ON metric_samples(ts_unix DESC);`,
	}

	for _, q := range queries {
		if _, err := db.conn.Exec(q); err != nil {
			return err
		}
	}

	// Handle migration from old allowlist system to new password-only system
	if err := db.migrateFromAllowlist(); err != nil {
		fmt.Println(msg.LogFailed(msg.ComponentDatabase, "migration from allowlist", err.Error()))
	}

	return nil
}

// migrateFromAllowlist handles migration from old allowlist system to new password-only system
func (db *DB) migrateFromAllowlist() error {
	// Check if old schema exists (has is_allowed column)
	var hasOldSchema bool
	err := db.conn.QueryRow(`
		SELECT COUNT(*) > 0 FROM pragma_table_info('users') WHERE name = 'is_allowed'
	`).Scan(&hasOldSchema)
	if err != nil {
		return err
	}

	if !hasOldSchema {
		// New schema already in place
		return nil
	}

	// Check if new columns already exist
	var hasNewSchema bool
	err = db.conn.QueryRow(`
		SELECT COUNT(*) > 0 FROM pragma_table_info('users') WHERE name = 'first_auth_at'
	`).Scan(&hasNewSchema)
	if err != nil {
		return err
	}

	if hasNewSchema {
		// Migration already completed
		return nil
	}

	// Add new columns for password-only system
	migrationQueries := []string{
		`ALTER TABLE users ADD COLUMN first_auth_at TEXT`,
		`ALTER TABLE users ADD COLUMN last_auth_at TEXT`,
		`ALTER TABLE users ADD COLUMN created_at TEXT DEFAULT CURRENT_TIMESTAMP`,
	}

	for _, q := range migrationQueries {
		if _, err := db.conn.Exec(q); err != nil {
			return err
		}
	}

	// Migrate existing allowed users to new system
	_, err = db.conn.Exec(`
		UPDATE users SET 
			first_auth_at = joined_at,
			last_auth_at = last_seen_at,
			created_at = joined_at
		WHERE is_allowed = 1
	`)
	if err != nil {
		return err
	}

	// Migrate existing non-allowed users to new system (keep their data but no auth history)
	_, err = db.conn.Exec(`
		UPDATE users SET 
			created_at = COALESCE(joined_at, datetime('now'))
		WHERE is_allowed = 0 OR is_allowed IS NULL
	`)
	if err != nil {
		return err
	}

	fmt.Println(msg.LogCompleted(msg.ComponentDatabase, "migration from allowlist to password-only system completed"))
	return nil
}

// User methods

func (db *DB) GetUser(id int64) (*User, error) {
	var u User
	var firstAuthAt, lastAuthAt, createdAt string
	var alertsEnabled int

	err := db.conn.QueryRow(`
		SELECT telegram_user_id, username, first_name, last_name, first_auth_at, last_auth_at, alerts_enabled, created_at
		FROM users WHERE telegram_user_id = ?`, id).Scan(
		&u.ID, &u.Username, &u.FirstName, &u.LastName, &firstAuthAt, &lastAuthAt, &alertsEnabled, &createdAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	u.FirstAuthAt, _ = time.Parse(time.RFC3339, firstAuthAt)
	u.LastAuthAt, _ = time.Parse(time.RFC3339, lastAuthAt)
	u.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	u.AlertsEnabled = alertsEnabled == 1

	return &u, nil
}

func (db *DB) UpdateUser(u *User) error {
	alertsEnabled := 0
	if u.AlertsEnabled {
		alertsEnabled = 1
	}

	_, err := db.conn.Exec(`
		INSERT INTO users (telegram_user_id, username, first_name, last_name, first_auth_at, last_auth_at, alerts_enabled, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(telegram_user_id) DO UPDATE SET
			username = excluded.username,
			first_name = excluded.first_name,
			last_name = excluded.last_name,
			first_auth_at = excluded.first_auth_at,
			last_auth_at = excluded.last_auth_at,
			alerts_enabled = excluded.alerts_enabled,
			created_at = excluded.created_at`,
		u.ID, u.Username, u.FirstName, u.LastName,
		u.FirstAuthAt.Format(time.RFC3339), u.LastAuthAt.Format(time.RFC3339), alertsEnabled,
		u.CreatedAt.Format(time.RFC3339))
	return err
}

func (db *DB) DeleteUser(id int64) error {
	_, err := db.conn.Exec("DELETE FROM users WHERE telegram_user_id = ?", id)
	return err
}

func (db *DB) GetAllowedUsers() ([]User, error) {
	rows, err := db.conn.Query(`SELECT telegram_user_id, alerts_enabled, first_auth_at FROM users ORDER BY last_auth_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var u User
		var alertsEnabled int
		var firstAuthAt string
		if err := rows.Scan(&u.ID, &alertsEnabled, &firstAuthAt); err != nil {
			return nil, err
		}
		u.FirstAuthAt, _ = time.Parse(time.RFC3339, firstAuthAt)
		u.AlertsEnabled = alertsEnabled == 1
		users = append(users, u)
	}
	return users, nil
}

// Settings methods

func (db *DB) GetSetting(key string) (string, error) {
	var val string
	err := db.conn.QueryRow("SELECT value FROM settings WHERE key = ?", key).Scan(&val)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return val, err
}

func (db *DB) SetSetting(key, value string) error {
	_, err := db.conn.Exec("INSERT INTO settings (key, value) VALUES (?, ?) ON CONFLICT(key) DO UPDATE SET value = excluded.value", key, value)
	return err
}

// Alert state methods

func (db *DB) GetAlertState(key string) (*AlertState, error) {
	var s AlertState
	var isActive int
	err := db.conn.QueryRow("SELECT alert_key, is_active, active_since_unix, last_triggered_unix, last_recovered_unix FROM alert_state WHERE alert_key = ?", key).Scan(
		&s.Key, &isActive, &s.ActiveSinceUnix, &s.LastTriggeredUnix, &s.LastRecoveredUnix)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	s.IsActive = isActive == 1
	return &s, nil
}

func (db *DB) UpdateAlertState(s *AlertState) error {
	return db.executeWithRetry(context.Background(), func() error {
		isActive := 0
		if s.IsActive {
			isActive = 1
		}
		_, err := db.conn.Exec(`
			INSERT INTO alert_state (alert_key, is_active, active_since_unix, last_triggered_unix, last_recovered_unix)
			VALUES (?, ?, ?, ?, ?)
			ON CONFLICT(alert_key) DO UPDATE SET
				is_active = excluded.is_active,
				active_since_unix = excluded.active_since_unix,
				last_triggered_unix = excluded.last_triggered_unix,
				last_recovered_unix = excluded.last_recovered_unix`,
			s.Key, isActive, s.ActiveSinceUnix, s.LastTriggeredUnix, s.LastRecoveredUnix)
		return err
	})
}

// Failed auth methods

func (db *DB) GetFailedAuth(userID int64) (int, int64, error) {
	var count int
	var lastFail int64
	err := db.conn.QueryRow("SELECT fail_count, last_fail_unix FROM failed_auth WHERE telegram_user_id = ?", userID).Scan(&count, &lastFail)
	if err == sql.ErrNoRows {
		return 0, 0, nil
	}
	return count, lastFail, err
}

func (db *DB) IncrementFailedAuth(userID int64) error {
	_, err := db.conn.Exec(`
		INSERT INTO failed_auth (telegram_user_id, fail_count, last_fail_unix)
		VALUES (?, 1, ?)
		ON CONFLICT(telegram_user_id) DO UPDATE SET
			fail_count = fail_count + 1,
			last_fail_unix = excluded.last_fail_unix`,
		userID, time.Now().Unix())
	return err
}

func (db *DB) ResetFailedAuth(userID int64) error {
	_, err := db.conn.Exec("DELETE FROM failed_auth WHERE telegram_user_id = ?", userID)
	return err
}

// Metric samples methods

func (db *DB) SaveMetricSample(s *metrics.MetricSample) error {
	return db.executeWithRetry(context.Background(), func() error {
		// Round values to 2 decimal places
		cpu := round(s.CPUPercent)
		mem := round(s.MemPercent)
		swap := round(s.SwapPercent)
		disk := round(s.DiskPercent)
		l1 := round(s.Load1)
		l5 := round(s.Load5)
		l15 := round(s.Load15)
		uptime := round(s.Uptime)

		_, err := db.conn.Exec(`
			INSERT INTO metric_samples (ts_unix, cpu_percent, mem_percent, swap_percent, disk_percent, load1, load5, load15, uptime)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			s.Timestamp.Unix(), cpu, mem, swap, disk, l1, l5, l15, uptime)
		if err != nil {
			return err
		}

		for _, v := range s.Volumes {
			_, err = db.conn.Exec(`
				INSERT INTO volume_samples (ts_unix, mount_path, disk_percent)
				VALUES (?, ?, ?)`,
				s.Timestamp.Unix(), v.Path, round(v.Percent))
			if err != nil {
				return fmt.Errorf("failed to save volume sample for %s: %w", v.Path, err)
			}
		}
		return nil
	})
}

func round(val float64) float64 {
	return math.Round(val*100) / 100
}

func (db *DB) GetRecentSamples(n int) (int, error) {
	var count int
	err := db.conn.QueryRow("SELECT COUNT(*) FROM (SELECT 1 FROM metric_samples LIMIT ?)", n).Scan(&count)
	return count, err
}

func (db *DB) GetLatestMetric() (*metrics.MetricSample, error) {
	var s metrics.MetricSample
	var tsUnix int64

	err := db.conn.QueryRow(`
		SELECT ts_unix, cpu_percent, mem_percent, swap_percent, disk_percent, load1, load5, load15, uptime
		FROM metric_samples 
		ORDER BY ts_unix DESC 
		LIMIT 1
	`).Scan(&tsUnix, &s.CPUPercent, &s.MemPercent, &s.SwapPercent, &s.DiskPercent, &s.Load1, &s.Load5, &s.Load15, &s.Uptime)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	s.Timestamp = time.Unix(tsUnix, 0)

	// Load volume samples for the same timestamp
	rows, err := db.conn.Query(`
		SELECT mount_path, disk_percent FROM volume_samples
		WHERE ts_unix = ?
		ORDER BY mount_path`, tsUnix)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var v metrics.VolumeSample
			if scanErr := rows.Scan(&v.Path, &v.Percent); scanErr == nil {
				s.Volumes = append(s.Volumes, v)
			}
		}
	}

	return &s, nil
}

// GetLatestVolumePercent returns the most recently recorded disk percent for
// the given mount path, plus a boolean indicating whether a row was found.
func (db *DB) GetLatestVolumePercent(path string) (float64, bool, error) {
	var pct float64
	err := db.conn.QueryRow(`
		SELECT disk_percent FROM volume_samples
		WHERE mount_path = ?
		ORDER BY ts_unix DESC
		LIMIT 1`, path).Scan(&pct)
	if err == sql.ErrNoRows {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, err
	}
	return pct, true, nil
}

// Alert transition methods

type Alert struct {
	ID         int64
	AlertType  string
	Severity   string
	Value      float64
	Threshold  float64
	Message    string
	Transition string
	Timestamp  time.Time
	SentAt     *time.Time
	CreatedAt  time.Time
}

func (db *DB) StoreAlert(alertType, severity string, value, threshold float64, message, transition string, timestamp time.Time) error {
	return db.executeWithRetry(context.Background(), func() error {
		_, err := db.conn.Exec(`
			INSERT INTO alerts (alert_type, severity, value, threshold, message, transition, timestamp_unix)
			VALUES (?, ?, ?, ?, ?, ?, ?)
		`, alertType, severity, value, threshold, message, transition, timestamp.Unix())
		return err
	})
}

func (db *DB) GetUnsentAlerts(limit int) ([]Alert, error) {
	query := `
		SELECT id, alert_type, severity, value, threshold, message, transition, timestamp_unix, sent_at_unix, created_at
		FROM alerts 
		WHERE sent_at_unix IS NULL 
		ORDER BY timestamp_unix ASC
	`
	if limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", limit)
	}

	rows, err := db.conn.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var alerts []Alert
	for rows.Next() {
		var a Alert
		var tsUnix int64
		var sentAtUnix *int64
		var createdAt string
		err := rows.Scan(&a.ID, &a.AlertType, &a.Severity, &a.Value, &a.Threshold, &a.Message, &a.Transition, &tsUnix, &sentAtUnix, &createdAt)
		if err != nil {
			return nil, err
		}

		a.Timestamp = time.Unix(tsUnix, 0)
		if sentAtUnix != nil && *sentAtUnix > 0 {
			sentAt := time.Unix(*sentAtUnix, 0)
			a.SentAt = &sentAt
		}
		a.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		alerts = append(alerts, a)
	}

	return alerts, nil
}

func (db *DB) MarkAlertSent(id int64) error {
	return db.executeWithRetry(context.Background(), func() error {
		_, err := db.conn.Exec(`
			UPDATE alerts
			SET sent_at_unix = ?
			WHERE id = ?
		`, time.Now().Unix(), id)
		return err
	})
}

// PurgeOldData deletes metric_samples and alerts older than retentionDays days.
// Returns the total number of rows deleted.
func (db *DB) PurgeOldData(retentionDays int) (int64, error) {
	cutoff := time.Now().AddDate(0, 0, -retentionDays).Unix()

	var total int64
	for _, query := range []string{
		`DELETE FROM metric_samples WHERE ts_unix < ?`,
		`DELETE FROM volume_samples WHERE ts_unix < ?`,
		`DELETE FROM alerts WHERE timestamp_unix < ?`,
	} {
		result, err := db.conn.Exec(query, cutoff)
		if err != nil {
			return total, err
		}
		n, _ := result.RowsAffected()
		total += n
	}
	return total, nil
}
