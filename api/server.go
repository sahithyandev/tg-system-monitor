package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"tg-system-monitor/db"
	"tg-system-monitor/message"
	"tg-system-monitor/metrics"
)

// Server exposes system metrics over HTTP for consumption by other services.
type Server struct {
	db          *db.DB
	addr        string
	volumeSizes []metrics.VolumeInfo
}

// NewServer creates a new metrics API server.
func NewServer(database *db.DB, addr string, volumeSizes []metrics.VolumeInfo) *Server {
	return &Server{db: database, addr: addr, volumeSizes: volumeSizes}
}

// Start begins listening for requests and blocks until ctx is cancelled.
func (s *Server) Start(ctx context.Context) {
	mux := http.NewServeMux()
	mux.HandleFunc("/metrics", s.handleMetrics)
	mux.HandleFunc("/metrics/history", s.handleHistory)
	mux.HandleFunc("/health", s.handleHealth)

	srv := &http.Server{
		Addr:    s.addr,
		Handler: mux,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("%s", message.LogFailed(message.ComponentAPI, "http server", err.Error()))
		}
	}()

	<-ctx.Done()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("%s", message.LogFailed(message.ComponentAPI, "graceful shutdown", err.Error()))
	}
}

type errorResponse struct {
	Error string `json:"error"`
}

type metricsResponse struct {
	Timestamp   int64        `json:"timestamp"`
	UptimeSecs  float64      `json:"uptime_seconds"`
	CPUPercent  float64      `json:"cpu_percent"`
	MemPercent  float64      `json:"mem_percent"`
	SwapPercent float64      `json:"swap_percent"`
	DiskPercent float64      `json:"disk_percent"`
	DiskTotal   uint64       `json:"disk_total_bytes"`
	Load1       float64      `json:"load1"`
	Load5       float64      `json:"load5"`
	Load15      float64      `json:"load15"`
	Volumes     []volumeJSON `json:"volumes"`
}

type volumeJSON struct {
	Path       string  `json:"path"`
	Percent    float64 `json:"percent"`
	TotalBytes uint64  `json:"total_bytes,omitempty"`
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("%s", message.LogFailed(message.ComponentAPI, "json encode", err.Error()))
	}
}

func (s *Server) handleMetrics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, errorResponse{Error: "method not allowed"})
		return
	}

	sample, err := s.db.GetLatestMetric()
	if err != nil {
		log.Printf("%s", message.LogFailed(message.ComponentAPI, "get latest metric", err.Error()))
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: fmt.Sprintf("failed to retrieve metrics: %s", err.Error())})
		return
	}

	if sample == nil {
		writeJSON(w, http.StatusServiceUnavailable, errorResponse{Error: message.NoMetricsAvailable})
		return
	}

	sizeByPath := make(map[string]uint64, len(s.volumeSizes))
	for _, vi := range s.volumeSizes {
		sizeByPath[vi.Path] = vi.TotalBytes
	}

	resp := metricsResponse{
		Timestamp:   sample.Timestamp.Unix(),
		UptimeSecs:  sample.Uptime,
		CPUPercent:  sample.CPUPercent,
		MemPercent:  sample.MemPercent,
		SwapPercent: sample.SwapPercent,
		DiskPercent: sample.DiskPercent,
		DiskTotal:   sizeByPath["/"],
		Load1:       sample.Load1,
		Load5:       sample.Load5,
		Load15:      sample.Load15,
		Volumes:     make([]volumeJSON, 0, len(sample.Volumes)),
	}
	for _, v := range sample.Volumes {
		resp.Volumes = append(resp.Volumes, volumeJSON{
			Path:       v.Path,
			Percent:    v.Percent,
			TotalBytes: sizeByPath[v.Path],
		})
	}

	writeJSON(w, http.StatusOK, resp)
}

type historyVolume struct {
	Path       string  `json:"path"`
	Percent    db.Stat `json:"percent"`
	TotalBytes uint64  `json:"total_bytes,omitempty"`
}

type historyPoint struct {
	Timestamp int64           `json:"timestamp"`
	CPU       db.Stat         `json:"cpu_percent"`
	Mem       db.Stat         `json:"mem_percent"`
	Swap      db.Stat         `json:"swap_percent"`
	Disk      db.Stat         `json:"disk_percent"`
	Load1     db.Stat         `json:"load1"`
	Load5     db.Stat         `json:"load5"`
	Load15    db.Stat         `json:"load15"`
	Volumes   []historyVolume `json:"volumes"`
}

type historyResponse struct {
	From       int64          `json:"from"`
	To         int64          `json:"to"`
	BucketSecs int64          `json:"bucket_seconds"`
	Points     []historyPoint `json:"points"`
}

const (
	// historyMaxPoints caps the buckets one response may contain. A 3-month
	// range at the ~15s sample rate is ~520k rows; downsampling keeps it small.
	historyMaxPoints = 5000
	// historyMinBucket is the finest resolution offered (the sample interval).
	historyMinBucket = 15
	// historyTargetPoints is how many buckets an auto-picked resolution aims for.
	historyTargetPoints = 2000
)

// handleHistory returns downsampled metric history for charting.
// Query params: from, to (unix seconds; default: last 1h); bucket (seconds per
// aggregation bucket; default: auto-picked from the range).
func (s *Server) handleHistory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, errorResponse{Error: "method not allowed"})
		return
	}

	q := r.URL.Query()
	now := time.Now().Unix()
	from, to := now-3600, now
	if v := q.Get("from"); v != "" {
		n, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid 'from': want unix seconds"})
			return
		}
		from = n
	}
	if v := q.Get("to"); v != "" {
		n, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid 'to': want unix seconds"})
			return
		}
		to = n
	}
	if from > to {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "'from' must not be after 'to'"})
		return
	}

	span := to - from
	var bucket int64
	if v := q.Get("bucket"); v != "" {
		n, err := strconv.ParseInt(v, 10, 64)
		if err != nil || n <= 0 {
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid 'bucket': want positive seconds"})
			return
		}
		bucket = n
	} else {
		// Round the span/target up to a whole number of seconds, floored at the
		// sample interval.
		bucket = (span + historyTargetPoints - 1) / historyTargetPoints
		if bucket < historyMinBucket {
			bucket = historyMinBucket
		}
	}

	if span/bucket > historyMaxPoints {
		writeJSON(w, http.StatusBadRequest, errorResponse{
			Error: fmt.Sprintf("range too wide for bucket=%ds (max %d points); use a larger bucket", bucket, historyMaxPoints),
		})
		return
	}

	buckets, err := s.db.GetMetricHistory(from, to, bucket)
	if err != nil {
		log.Printf("%s", message.LogFailed(message.ComponentAPI, "get metric history", err.Error()))
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: fmt.Sprintf("failed to retrieve history: %s", err.Error())})
		return
	}

	sizeByPath := make(map[string]uint64, len(s.volumeSizes))
	for _, vi := range s.volumeSizes {
		sizeByPath[vi.Path] = vi.TotalBytes
	}

	resp := historyResponse{From: from, To: to, BucketSecs: bucket, Points: make([]historyPoint, 0, len(buckets))}
	for _, b := range buckets {
		p := historyPoint{
			Timestamp: b.BucketStart,
			CPU:       b.CPU,
			Mem:       b.Mem,
			Swap:      b.Swap,
			Disk:      b.Disk,
			Load1:     b.Load1,
			Load5:     b.Load5,
			Load15:    b.Load15,
			Volumes:   make([]historyVolume, 0, len(b.Volumes)),
		}
		for _, v := range b.Volumes {
			p.Volumes = append(p.Volumes, historyVolume{
				Path:       v.Path,
				Percent:    v.Percent,
				TotalBytes: sizeByPath[v.Path],
			})
		}
		resp.Points = append(resp.Points, p)
	}
	writeJSON(w, http.StatusOK, resp)
}

type healthResponse struct {
	Status string `json:"status"`
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, errorResponse{Error: "method not allowed"})
		return
	}

	if err := s.db.Ping(); err != nil {
		writeJSON(w, http.StatusServiceUnavailable, healthResponse{Status: "unhealthy"})
		return
	}

	writeJSON(w, http.StatusOK, healthResponse{Status: "ok"})
}
