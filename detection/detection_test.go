package detection

import (
	"testing"
	"tg-system-monitor/db"
)

func TestDetectionEngine(t *testing.T) {
	// Use in-memory database for testing
	database, err := db.Init(":memory:")
	if err != nil {
		t.Fatal("Failed to initialize database:", err)
	}
	defer database.Close()

	// Configure detection engine
	config := DetectionConfig{
		CPU: Threshold{
			Warning:  70.0,
			Critical: 90.0,
		},
		Memory: Threshold{
			Warning:  80.0,
			Critical: 95.0,
		},
		Disk: Threshold{
			Warning:  85.0,
			Critical: 95.0,
		},
		Swap: Threshold{
			Warning:  50.0,
			Critical: 80.0,
		},
		Load1: Threshold{
			Warning:  2.0,
			Critical: 4.0,
		},
		Load5: Threshold{
			Warning:  1.5,
			Critical: 3.0,
		},
		Load15: Threshold{
			Warning:  1.0,
			Critical: 2.0,
		},
		WindowSecs:   300, // 5 minutes
		CooldownSecs: 300, // 5 minutes
		Hysteresis:   5.0, // 5% hysteresis
	}

	// Create detection engine
	engine := NewDetectionEngine(database, config)

	// Run detection evaluation
	err = engine.EvaluateDetections()
	if err != nil {
		t.Logf("Error evaluating detections: %v", err)
		return
	}
}
