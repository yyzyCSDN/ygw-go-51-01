package main

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"deviceshadow/internal/desired"
	"deviceshadow/internal/device"
	"deviceshadow/internal/notify"
	"deviceshadow/internal/report"
	"deviceshadow/internal/shadow"
	"deviceshadow/internal/sync"
)

// ServerDeps bundles every component the HTTP layer needs.
type ServerDeps struct {
	Store       *shadow.Store
	Registry    *device.Registry
	Allocator   *sync.VersionAllocator
	Sync        *sync.Service
	Report      *report.Service
	Desired     *desired.Service
	Batch       *desired.BatchService
	Hub         *notify.Hub
	ConsolePath string
}

// Server exposes the REST and console routes.
type Server struct {
	deps ServerDeps
	mux  *http.ServeMux
}

// NewServer builds the HTTP server.
func NewServer(deps ServerDeps) *Server {
	s := &Server{deps: deps, mux: http.NewServeMux()}
	s.routes()
	return s
}

// Handler returns the root HTTP handler.
func (s *Server) Handler() http.Handler {
	return s.withMiddleware(s.mux)
}

func (s *Server) routes() {
	s.mux.HandleFunc("GET /healthz", s.handleHealth)
	s.mux.HandleFunc("GET /", s.handleConsole)
	s.mux.HandleFunc("GET /api/v1/overview", s.handleOverview)
	s.mux.HandleFunc("GET /api/v1/devices", s.handleListDevices)
	s.mux.HandleFunc("POST /api/v1/devices", s.handleRegisterDevice)
	s.mux.HandleFunc("GET /api/v1/devices/{id}/shadow", s.handleGetShadow)
	s.mux.HandleFunc("GET /api/v1/devices/{id}/shadow/history", s.handleShadowHistory)
	s.mux.HandleFunc("POST /api/v1/devices/{id}/report", s.handleReport)
	s.mux.HandleFunc("POST /api/v1/devices/{id}/desired", s.handleSetDesired)
	s.mux.HandleFunc("POST /api/v1/devices/{id}/desired/batch", s.handleBatchDesired)
	s.mux.HandleFunc("POST /api/v1/devices/{id}/desired/transaction", s.handleTransactionDesired)
	s.mux.HandleFunc("POST /api/v1/batches/desired", s.handleBatchDesired)
	s.mux.HandleFunc("DELETE /api/v1/devices/{id}/shadow", s.handleDeleteShadow)
	s.mux.HandleFunc("POST /api/v1/devices/{id}/reconnect", s.handleReconnect)
	s.mux.HandleFunc("POST /api/v1/devices/{id}/offline", s.handleOffline)
	s.mux.HandleFunc("POST /api/v1/devices/{id}/ack", s.handleAck)
	s.mux.HandleFunc("POST /api/v1/devices/{id}/subscribe", s.handleSubscribe)
	s.mux.HandleFunc("POST /api/v1/devices/{id}/unsubscribe", s.handleUnsubscribe)
	s.mux.HandleFunc("GET /api/v1/batches", s.handleBatches)
}

func (s *Server) withMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		next.ServeHTTP(w, r)
		log.Printf("%s %s %s", r.Method, r.URL.Path, time.Since(started).Round(time.Microsecond))
	})
}

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
