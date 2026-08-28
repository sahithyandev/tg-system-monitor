package db

import (
	"path/filepath"
	"testing"
	"time"

	"tg-system-monitor/metrics"
)

func TestGetMetricHistory(t *testing.T) {
	d, err := Init(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	if err := d.Migrate(); err != nil {
		t.Fatal(err)
	}

	// Anchor to a bucket boundary so sample placement is predictable.
	base := time.Unix(1_700_000_100, 0) // multiple of 180
	// 6 samples one minute apart: cpu 0..5, volume "/" pct = cpu+10.
	for i := 0; i < 6; i++ {
		s := &metrics.MetricSample{
			Timestamp:  base.Add(time.Duration(i) * time.Minute),
			CPUPercent: float64(i),
			Volumes:    []metrics.VolumeSample{{Path: "/", Percent: float64(i) + 10}},
		}
		if err := d.SaveMetricSample(s); err != nil {
			t.Fatal(err)
		}
	}

	// 3-minute buckets over the whole range → 2 buckets.
	got, err := d.GetMetricHistory(base.Unix(), base.Add(6*time.Minute).Unix(), 180)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("want 2 buckets, got %d", len(got))
	}

	// Bucket 0: samples 0,1,2 → avg 1, max 2.
	if got[0].CPU.Avg != 1 || got[0].CPU.Max != 2 {
		t.Fatalf("bucket 0 cpu: got avg=%v max=%v", got[0].CPU.Avg, got[0].CPU.Max)
	}
	if len(got[0].Volumes) != 1 || got[0].Volumes[0].Path != "/" ||
		got[0].Volumes[0].Percent.Avg != 11 || got[0].Volumes[0].Percent.Max != 12 {
		t.Fatalf("bucket 0 volumes: %+v", got[0].Volumes)
	}

	// Bucket 1: samples 3,4,5 → avg 4, max 5.
	if got[1].CPU.Avg != 4 || got[1].CPU.Max != 5 {
		t.Fatalf("bucket 1 cpu: got avg=%v max=%v", got[1].CPU.Avg, got[1].CPU.Max)
	}

	// Oldest first.
	if got[0].BucketStart >= got[1].BucketStart {
		t.Fatalf("buckets not oldest-first: %d then %d", got[0].BucketStart, got[1].BucketStart)
	}

	if _, err := d.GetMetricHistory(base.Unix(), base.Unix(), 0); err == nil {
		t.Fatal("expected error for non-positive bucket")
	}
}
