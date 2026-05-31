package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"tg-system-monitor/db"
	"tg-system-monitor/message"
)

// Server exposes system metrics over HTTP for consumption by other services.
type Server struct {
	db   *db.DB
	addr string
}

// NewServer creates a new metrics API server.
func NewServer(database *db.DB, addr string) *Server {
	return &Server{db: database, addr: addr}
}

// Start begins listening for requests and blocks until ctx is cancelled.
func (s *Server) Start(ctx context.Context) {
	mux := http.NewServeMux()
	mux.HandleFunc("/metrics", s.handleMetrics)
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
	Load1       float64      `json:"load1"`
	Load5       float64      `json:"load5"`
	Load15      float64      `json:"load15"`
	Volumes     []volumeJSON `json:"volumes"`
}

type volumeJSON struct {
	Path    string  `json:"path"`
	Percent float64 `json:"percent"`
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

	resp := metricsResponse{
		Timestamp:   sample.Timestamp.Unix(),
		UptimeSecs:  sample.Uptime,
		CPUPercent:  sample.CPUPercent,
		MemPercent:  sample.MemPercent,
		SwapPercent: sample.SwapPercent,
		DiskPercent: sample.DiskPercent,
		Load1:       sample.Load1,
		Load5:       sample.Load5,
		Load15:      sample.Load15,
		Volumes:     make([]volumeJSON, 0, len(sample.Volumes)),
	}
	for _, v := range sample.Volumes {
		resp.Volumes = append(resp.Volumes, volumeJSON{Path: v.Path, Percent: v.Percent})
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
