package db

import (
	"database/sql"
	"fmt"
	"math"
	"time"
	"tg-system-monitor/metrics"

	_ "modernc.org/sqlite"
)

type User struct {
	ID            int64
	Username      string
	FirstName     string
	LastName      string
	JoinedAt      time.Time
	IsAllowed     bool
	AlertsEnabled bool
	LastSeenAt    time.Time
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
	conn, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	db := &DB{conn: conn}
	if err := db.migrate(); err != nil {
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}

	return db, nil
}

func (db *DB) Close() error {
	return db.conn.Close()
}

func (db *DB) Ping() error {
	return db.conn.Ping()
}

func (db *DB) migrate() error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS users (
			telegram_user_id INTEGER PRIMARY KEY,
			username TEXT,
			first_name TEXT,
			last_name TEXT,
			joined_at TEXT,
			is_allowed INTEGER NOT NULL DEFAULT 0,
			alerts_enabled INTEGER NOT NULL DEFAULT 1,
			last_seen_at TEXT
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
			load15 REAL
		);`,
	}

	for _, q := range queries {
		if _, err := db.conn.Exec(q); err != nil {
			return err
		}
	}
	return nil
}

// User methods

func (db *DB) GetUser(id int64) (*User, error) {
	var u User
	var joinedAt, lastSeenAt string
	var isAllowed, alertsEnabled int

	err := db.conn.QueryRow(`
		SELECT telegram_user_id, username, first_name, last_name, joined_at, is_allowed, alerts_enabled, last_seen_at
		FROM users WHERE telegram_user_id = ?`, id).Scan(
		&u.ID, &u.Username, &u.FirstName, &u.LastName, &joinedAt, &isAllowed, &alertsEnabled, &lastSeenAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	u.JoinedAt, _ = time.Parse(time.RFC3339, joinedAt)
	u.LastSeenAt, _ = time.Parse(time.RFC3339, lastSeenAt)
	u.IsAllowed = isAllowed == 1
	u.AlertsEnabled = alertsEnabled == 1

	return &u, nil
}

func (db *DB) UpdateUser(u *User) error {
	isAllowed := 0
	if u.IsAllowed {
		isAllowed = 1
	}
	alertsEnabled := 0
	if u.AlertsEnabled {
		alertsEnabled = 1
	}

	_, err := db.conn.Exec(`
		INSERT INTO users (telegram_user_id, username, first_name, last_name, joined_at, is_allowed, alerts_enabled, last_seen_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(telegram_user_id) DO UPDATE SET
			username = excluded.username,
			first_name = excluded.first_name,
			last_name = excluded.last_name,
			is_allowed = excluded.is_allowed,
			alerts_enabled = excluded.alerts_enabled,
			last_seen_at = excluded.last_seen_at`,
		u.ID, u.Username, u.FirstName, u.LastName, u.JoinedAt.Format(time.RFC3339), isAllowed, alertsEnabled, u.LastSeenAt.Format(time.RFC3339))
	return err
}

func (db *DB) GetAllowedUsers() ([]User, error) {
	rows, err := db.conn.Query(`SELECT telegram_user_id, alerts_enabled FROM users WHERE is_allowed = 1`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var u User
		var alertsEnabled int
		if err := rows.Scan(&u.ID, &alertsEnabled); err != nil {
			return nil, err
		}
		u.IsAllowed = true
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
	// Round values to 2 decimal places
	cpu := round(s.CPUPercent)
	mem := round(s.MemPercent)
	swap := round(s.SwapPercent)
	disk := round(s.DiskPercent)
	l1 := round(s.Load1)
	l5 := round(s.Load5)
	l15 := round(s.Load15)

	_, err := db.conn.Exec(`
		INSERT INTO metric_samples (ts_unix, cpu_percent, mem_percent, swap_percent, disk_percent, load1, load5, load15)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		s.Timestamp.Unix(), cpu, mem, swap, disk, l1, l5, l15)
	return err
}

func round(val float64) float64 {
	return math.Round(val*100) / 100
}

func (db *DB) GetRecentSamples(n int) (int, error) {
	var count int
	err := db.conn.QueryRow("SELECT COUNT(*) FROM (SELECT 1 FROM metric_samples LIMIT ?)", n).Scan(&count)
	return count, err
}
