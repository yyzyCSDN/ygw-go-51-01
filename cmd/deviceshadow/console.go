package main

import (
	"context"
	"net/http"
	"os"
	"time"

	"deviceshadow/internal/desired"
	"deviceshadow/internal/shadow"
)

// handleConsole serves the browser console page with the right content type.
func (s *Server) handleConsole(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	data, err := os.ReadFile(s.deps.ConsolePath)
	if err != nil {
		writeError(w, http.StatusNotFound, "console page not found")
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write(data)
}

// gcLoop periodically drops expired tombstones.
func gcLoop(ctx context.Context, store *shadow.Store, ttl time.Duration) {
	ticker := time.NewTicker(ttl)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			cutoff := time.Now().UnixNano() - int64(ttl)
			_ = store.GCTombstones(cutoff)
		}
	}
}

// retryLoop re-delivers outstanding batch items on a fixed interval.
func retryLoop(ctx context.Context, batch *desired.BatchService, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			for _, id := range batch.PendingBatches() {
				batch.RetryOutstanding(id, time.Now().UnixNano())
			}
		}
	}
}
