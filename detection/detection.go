package detection

import (
	"database/sql"
	"fmt"
	"log"
	"tg-system-monitor/db"
	"tg-system-monitor/metrics"
	"time"

	_ "modernc.org/sqlite"
)

type Severity string

const (
	Warning  Severity = "warning"
	Critical Severity = "critical"
)

type AlertType string

const (
	CPU         AlertType = "cpu"
	Memory      AlertType = "memory"
	Disk        AlertType = "disk"
	Swap        AlertType = "swap"
	Load1       AlertType = "load1"
	Load5       AlertType = "load5"
	Load15      AlertType = "load15"
	CPUHigh     AlertType = "cpu_high"
	MemoryHigh  AlertType = "memory_high"
	DiskHigh    AlertType = "disk_high"
	SwapHigh    AlertType = "swap_high"
	LoadSpike   AlertType = "load_spike"
	ResourceLow AlertType = "resource_low"
)

type Alert struct {
	Type      AlertType
	Severity  Severity
	Value     float64
	Threshold float64
	Message   string
}

type AlertWithTransition struct {
	Alert      Alert
	Transition string // "triggered", "recovered"
	Timestamp  time.Time
}

type Threshold struct {
	Warning  float64
	Critical float64
}

type VolumeThreshold struct {
	Path      string
	Threshold Threshold
}

type DetectionConfig struct {
	CPU          Threshold
	Memory       Threshold
	Disk         Threshold
	Swap         Threshold
	Load1        Threshold
	Load5        Threshold
	Load15       Threshold
	Volumes      []VolumeThreshold
	WindowSecs   int
	CooldownSecs int
	Hysteresis   float64
}

type DetectionEngine struct {
	db     *db.DB
	config DetectionConfig
}

func NewDetectionEngine(database *db.DB, config DetectionConfig) *DetectionEngine {
	return &DetectionEngine{
		db:     database,
		config: config,
	}
}

func (d *DetectionEngine) EvaluateDetections() error {
	samples, err := d.getRecentSamples(d.config.WindowSecs)
	if err != nil {
		return fmt.Errorf("failed to get recent samples: %w", err)
	}

	if len(samples) < 2 {
		return nil
	}

	// Basic threshold alerts
	d.evaluateThresholdAlert(CPU, samples, d.config.CPU, func(s *metrics.MetricSample) float64 { return s.CPUPercent })
	d.evaluateThresholdAlert(Memory, samples, d.config.Memory, func(s *metrics.MetricSample) float64 { return s.MemPercent })
	d.evaluateThresholdAlert(Disk, samples, d.config.Disk, func(s *metrics.MetricSample) float64 { return s.DiskPercent })
	d.evaluateThresholdAlert(Swap, samples, d.config.Swap, func(s *metrics.MetricSample) float64 { return s.SwapPercent })
	d.evaluateThresholdAlert(Load1, samples, d.config.Load1, func(s *metrics.MetricSample) float64 { return s.Load1 })
	d.evaluateThresholdAlert(Load5, samples, d.config.Load5, func(s *metrics.MetricSample) float64 { return s.Load5 })
	d.evaluateThresholdAlert(Load15, samples, d.config.Load15, func(s *metrics.MetricSample) float64 { return s.Load15 })

	// Per-volume threshold alerts
	for _, v := range d.config.Volumes {
		d.evaluateVolumeAlert(v.Path, v.Threshold)
	}

	// Derived alerts
	d.evaluateDerivedAlerts(samples)

	return nil
}

func (d *DetectionEngine) getRecentSamples(windowSecs int) ([]*metrics.MetricSample, error) {
	cutoff := time.Now().Add(-time.Duration(windowSecs) * time.Second).Unix()

	// Use a more efficient query with LIMIT to avoid loading too much data
	query := `
		SELECT ts_unix, cpu_percent, mem_percent, swap_percent, disk_percent, load1, load5, load15, uptime
		FROM metric_samples 
		WHERE ts_unix >= ?
		ORDER BY ts_unix DESC
		LIMIT 100
	`

	rows, err := d.db.GetConn().Query(query, cutoff)
	if err != nil {
		return nil, fmt.Errorf("failed to query recent samples: %w", err)
	}
	defer rows.Close()

	var samples []*metrics.MetricSample
	for rows.Next() {
		var s metrics.MetricSample
		var tsUnix int64
		err := rows.Scan(&tsUnix, &s.CPUPercent, &s.MemPercent, &s.SwapPercent, &s.DiskPercent, &s.Load1, &s.Load5, &s.Load15, &s.Uptime)
		if err != nil {
			return nil, fmt.Errorf("failed to scan metric sample: %w", err)
		}
		s.Timestamp = time.Unix(tsUnix, 0)
		samples = append(samples, &s)
	}

	// Reverse to maintain chronological order
	for i, j := 0, len(samples)-1; i < j; i, j = i+1, j-1 {
		samples[i], samples[j] = samples[j], samples[i]
	}

	return samples, nil
}

func (d *DetectionEngine) evaluateValue(alertType AlertType, value float64, threshold Threshold) {
	warningKey := fmt.Sprintf("%s_warning", string(alertType))
	criticalKey := fmt.Sprintf("%s_critical", string(alertType))

	now := time.Now().Unix()

	// Check critical threshold
	if value >= threshold.Critical {
		d.checkAndStoreAlert(criticalKey, alertType, Critical, value, threshold.Critical, now)
	} else if value >= threshold.Warning {
		d.checkAndStoreAlert(warningKey, alertType, Warning, value, threshold.Warning, now)
	} else {
		// Check for recovery
		d.checkAndStoreRecovery(warningKey, alertType, Warning, value, threshold.Warning, now)
		d.checkAndStoreRecovery(criticalKey, alertType, Critical, value, threshold.Critical, now)
	}
}

func (d *DetectionEngine) evaluateThresholdAlert(alertType AlertType, samples []*metrics.MetricSample, threshold Threshold, getValue func(*metrics.MetricSample) float64) {
	latest := samples[len(samples)-1]
	d.evaluateValue(alertType, getValue(latest), threshold)
}

func (d *DetectionEngine) evaluateVolumeAlert(path string, threshold Threshold) {
	pct, ok, err := d.db.GetLatestVolumePercent(path)
	if err != nil {
		log.Printf("Error getting latest volume percent for %s: %v", path, err)
		return
	}
	if !ok {
		return
	}
	d.evaluateValue(AlertType("disk "+path), pct, threshold)
}

func (d *DetectionEngine) checkAndStoreAlert(key string, alertType AlertType, severity Severity, value, threshold float64, now int64) {
	state, err := d.db.GetAlertState(key)
	if err != nil {
		log.Printf("Error getting alert state for %s: %v", key, err)
		return
	}

	if state == nil {
		// New alert - store transition and create state
		message := fmt.Sprintf("%s %s: %.2f >= %.2f", alertType, severity, value, threshold)
		if err := d.db.StoreAlert(string(alertType), string(severity), value, threshold, message, "triggered", time.Unix(now, 0)); err != nil {
			log.Printf("Error storing alert transition for %s: %v", key, err)
			return
		}

		state = &db.AlertState{
			Key:               key,
			IsActive:          true,
			ActiveSinceUnix:   now,
			LastTriggeredUnix: now,
		}
		if err := d.db.UpdateAlertState(state); err != nil {
			log.Printf("Error updating alert state for %s: %v", key, err)
		}
		return
	}

	if !state.IsActive {
		// Reactivation after cooldown
		if now-state.LastRecoveredUnix >= int64(d.config.CooldownSecs) {
			message := fmt.Sprintf("%s %s: %.2f >= %.2f", alertType, severity, value, threshold)
			if err := d.db.StoreAlert(string(alertType), string(severity), value, threshold, message, "triggered", time.Unix(now, 0)); err != nil {
				log.Printf("Error storing alert transition for %s: %v", key, err)
				return
			}

			state.IsActive = true
			state.ActiveSinceUnix = now
			state.LastTriggeredUnix = now
			if err := d.db.UpdateAlertState(state); err != nil {
				log.Printf("Error updating alert state for %s: %v", key, err)
			}
		}
	} else if now-state.LastTriggeredUnix >= int64(d.config.CooldownSecs) {
		// Duplicate suppression - update trigger time but don't store transition
		state.LastTriggeredUnix = now
		d.db.UpdateAlertState(state)
	}
}

func (d *DetectionEngine) checkAndStoreRecovery(key string, alertType AlertType, severity Severity, value, threshold float64, now int64) {
	state, err := d.db.GetAlertState(key)
	if err != nil {
		log.Printf("Error getting alert state for %s: %v", key, err)
		return
	}

	if state != nil && state.IsActive {
		// Apply hysteresis
		recoveryThreshold := threshold - d.config.Hysteresis
		if value <= recoveryThreshold {
			message := fmt.Sprintf("%s %s recovered: %.2f <= %.2f", alertType, severity, value, recoveryThreshold)
			if err := d.db.StoreAlert(string(alertType), string(severity), value, threshold, message, "recovered", time.Unix(now, 0)); err != nil {
				log.Printf("Error storing alert transition for %s: %v", key, err)
				return
			}

			state.IsActive = false
			state.LastRecoveredUnix = now
			if err := d.db.UpdateAlertState(state); err != nil {
				log.Printf("Error updating alert state for %s: %v", key, err)
			}
		}
	}
}

func (d *DetectionEngine) evaluateDerivedAlerts(samples []*metrics.MetricSample) {
	if len(samples) < 2 {
		return
	}

	latest := samples[len(samples)-1]
	previous := samples[len(samples)-2]
	now := time.Now().Unix()

	// CPU sustained high alert
	if d.isSustainedHigh(samples, func(s *metrics.MetricSample) float64 { return s.CPUPercent }, d.config.CPU.Critical) {
		d.checkAndStoreAlert("cpu_high_critical", CPUHigh, Critical, latest.CPUPercent, d.config.CPU.Critical, now)
	} else {
		d.checkAndStoreRecovery("cpu_high_critical", CPUHigh, Critical, latest.CPUPercent, d.config.CPU.Critical, now)
	}

	// Memory sustained high alert
	if d.isSustainedHigh(samples, func(s *metrics.MetricSample) float64 { return s.MemPercent }, d.config.Memory.Critical) {
		d.checkAndStoreAlert("memory_high_critical", MemoryHigh, Critical, latest.MemPercent, d.config.Memory.Critical, now)
	} else {
		d.checkAndStoreRecovery("memory_high_critical", MemoryHigh, Critical, latest.MemPercent, d.config.Memory.Critical, now)
	}

	// Load spike detection
	loadIncrease := latest.Load1 - previous.Load1
	if loadIncrease >= 2.0 && latest.Load1 >= d.config.Load1.Warning {
		d.checkAndStoreAlert("load_spike_warning", LoadSpike, Warning, latest.Load1, previous.Load1, now)
	} else {
		d.checkAndStoreRecovery("load_spike_warning", LoadSpike, Warning, latest.Load1, previous.Load1, now)
	}

	// Resource pressure (multiple metrics high)
	pressureCount := 0
	if latest.CPUPercent >= d.config.CPU.Warning {
		pressureCount++
	}
	if latest.MemPercent >= d.config.Memory.Warning {
		pressureCount++
	}
	if latest.DiskPercent >= d.config.Disk.Warning {
		pressureCount++
	}

	if pressureCount >= 2 {
		d.checkAndStoreAlert("resource_low_warning", ResourceLow, Warning, float64(pressureCount), 2.0, now)
	} else {
		d.checkAndStoreRecovery("resource_low_warning", ResourceLow, Warning, float64(pressureCount), 2.0, now)
	}
}

func (d *DetectionEngine) isSustainedHigh(samples []*metrics.MetricSample, getValue func(*metrics.MetricSample) float64, threshold float64) bool {
	if len(samples) < 3 {
		return false
	}

	count := 0
	for _, sample := range samples {
		if getValue(sample) >= threshold {
			count++
		}
	}

	return float64(count)/float64(len(samples)) >= 0.8 // 80% of samples above threshold
}

// Helper method to get the underlying database connection
func (d *DetectionEngine) GetDB() *db.DB {
	return d.db
}

func (d *DetectionEngine) GetConn() *sql.DB {
	return d.db.GetConn()
}
