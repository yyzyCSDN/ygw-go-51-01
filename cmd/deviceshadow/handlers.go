package main

import (
	"encoding/json"
	"net/http"
	"time"

	"deviceshadow/internal/model"
	"deviceshadow/internal/notify"
)

type registerRequest struct {
	DeviceID string `json:"device_id"`
}

type reportRequest struct {
	Props    map[string]string `json:"props"`
	Version  int64             `json:"version"`
	ClientID string            `json:"client_id"`
}

type desiredRequest struct {
	Props  map[string]string `json:"props"`
	Source string            `json:"source"`
}

type batchRequest struct {
	Items []model.BatchItem `json:"items"`
}

type ackRequest struct {
	Version int64 `json:"version"`
}

type subRequest struct {
	SubID string `json:"sub_id"`
}

func (s *Server) handleRegisterDevice(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	now := time.Now().UnixNano()
	dev := s.deps.Registry.Register(req.DeviceID, now)
	writeJSON(w, http.StatusCreated, map[string]any{
		"device_id":  dev.ID,
		"slot":       dev.Slot,
		"registered": true,
	})
}

func (s *Server) handleListDevices(w http.ResponseWriter, _ *http.Request) {
	devices := s.deps.Registry.List()
	out := make([]map[string]any, 0, len(devices))
	for _, dev := range devices {
		doc := s.deps.Store.Get(dev.ID)
		out = append(out, map[string]any{
			"device_id":      dev.ID,
			"slot":           dev.Slot,
			"online":         dev.Online,
			"last_delivered": dev.LastDelivered,
			"last_ack":       dev.LastAck,
			"cached_ops":     dev.Cache.Size(),
			"shadow_version": versionOf(doc),
			"shadow_state":   string(s.deps.Store.StateOf(dev.ID)),
			"desired":        desiredOf(doc),
			"reported":       reportedOf(doc),
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"devices": out})
}

func (s *Server) handleGetShadow(w http.ResponseWriter, r *http.Request) {
	deviceID := r.PathValue("id")
	doc := s.deps.Store.Get(deviceID)
	if doc == nil {
		writeError(w, http.StatusNotFound, "shadow not found")
		return
	}
	writeJSON(w, http.StatusOK, doc)
}

func (s *Server) handleShadowHistory(w http.ResponseWriter, r *http.Request) {
	deviceID := r.PathValue("id")
	history := s.deps.Store.History(deviceID)
	changes := make([]map[string]any, 0, len(history))
	var previous *model.Document
	for _, event := range history {
		diff := notify.Diff(deviceID, previous, event.Snapshot)
		changes = append(changes, map[string]any{
			"type":        eventTypeName(event.Type),
			"version":     event.Version,
			"occurred_at": event.OccurredAt,
			"added":       diff.Added,
			"updated":     diff.Updated,
			"removed":     diff.Removed,
		})
		previous = event.Snapshot
	}
	writeJSON(w, http.StatusOK, map[string]any{"device_id": deviceID, "history": changes})
}

func eventTypeName(eventType model.EventType) string {
	switch eventType {
	case model.EventDesired:
		return "desired"
	case model.EventReport:
		return "report"
	case model.EventDelete:
		return "delete"
	case model.EventReconnect:
		return "reconnect"
	}
	return "unknown"
}

func (s *Server) handleReport(w http.ResponseWriter, r *http.Request) {
	deviceID := r.PathValue("id")
	var req reportRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	doc, err := s.deps.Report.HandleReport(deviceID, req.Props, req.Version, req.ClientID, time.Now().UnixNano())
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, doc)
}

func (s *Server) handleSetDesired(w http.ResponseWriter, r *http.Request) {
	deviceID := r.PathValue("id")
	var req desiredRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	source := req.Source
	if source == "" {
		source = "console"
	}
	doc, err := s.deps.Desired.ApplyDesired(deviceID, req.Props, source, time.Now().UnixNano())
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, doc)
}

func (s *Server) handleBatchDesired(w http.ResponseWriter, r *http.Request) {
	var req batchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	batch, err := s.deps.Batch.ApplyBatch(req.Items, time.Now().UnixNano())
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, batch)
}

func (s *Server) handleTransactionDesired(w http.ResponseWriter, r *http.Request) {
	deviceID := r.PathValue("id")
	var req desiredRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	doc, err := s.deps.Batch.ApplyTransactional(deviceID, req.Props, time.Now().UnixNano())
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, doc)
}

func (s *Server) handleDeleteShadow(w http.ResponseWriter, r *http.Request) {
	deviceID := r.PathValue("id")
	if err := s.deps.Store.Delete(deviceID, time.Now().UnixNano()); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"deleted": deviceID})
}

func (s *Server) handleReconnect(w http.ResponseWriter, r *http.Request) {
	deviceID := r.PathValue("id")
	s.deps.Registry.SetOnline(deviceID, true)
	ops := s.deps.Sync.Resume(deviceID)
	writeJSON(w, http.StatusOK, map[string]any{
		"device_id":       deviceID,
		"replayed_ops":    len(ops),
		"current_version": s.deps.Store.CurrentVersion(deviceID),
		"tombstoned":      s.deps.Store.IsTombstoned(deviceID),
	})
}

func (s *Server) handleOffline(w http.ResponseWriter, r *http.Request) {
	deviceID := r.PathValue("id")
	s.deps.Registry.SetOnline(deviceID, false)
	writeJSON(w, http.StatusOK, map[string]string{"device_id": deviceID, "online": "false"})
}

func (s *Server) handleAck(w http.ResponseWriter, r *http.Request) {
	deviceID := r.PathValue("id")
	var req ackRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	dev, ok := s.deps.Registry.Get(deviceID)
	if !ok {
		writeError(w, http.StatusNotFound, "device not found")
		return
	}
	dev.Ack(req.Version)
	writeJSON(w, http.StatusOK, map[string]any{"device_id": deviceID, "last_ack": dev.AckWatermark()})
}

func (s *Server) handleSubscribe(w http.ResponseWriter, r *http.Request) {
	deviceID := r.PathValue("id")
	subID, _ := s.deps.Hub.Subscribe(deviceID, 8)
	writeJSON(w, http.StatusCreated, map[string]string{"sub_id": subID, "topic": "shadow/" + deviceID})
}

func (s *Server) handleUnsubscribe(w http.ResponseWriter, r *http.Request) {
	deviceID := r.PathValue("id")
	var req subRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	s.deps.Hub.Unsubscribe(deviceID, req.SubID)
	writeJSON(w, http.StatusOK, map[string]string{"unsubscribed": req.SubID})
}

func (s *Server) handleBatches(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"pending_batches": s.deps.Batch.PendingBatches()})
}

func (s *Server) handleOverview(w http.ResponseWriter, _ *http.Request) {
	shards := make(map[uint64]int)
	for slot := uint64(0); slot < uint64(s.deps.Registry.Shards()); slot++ {
		shards[slot] = len(s.deps.Registry.BySlot(slot))
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"devices":         s.deps.Registry.Count(),
		"shadows":         len(s.deps.Store.List()),
		"tombstones":      s.deps.Store.TombstoneCount(),
		"subscribers":     len(s.deps.Hub.DevicesWithSubscribers()),
		"pending_batches": len(s.deps.Batch.PendingBatches()),
		"sync_metrics":    s.deps.Sync.Metrics(),
		"report_total":    s.deps.Report.ReportTotal(),
		"report_clients":  s.deps.Report.ClientStats(),
		"shard_devices":   shards,
		"topic_buckets":   s.deps.Hub.ShardDistribution(),
	})
}

func versionOf(doc *model.Document) int64 {
	if doc == nil {
		return 0
	}
	return doc.Version
}

func desiredOf(doc *model.Document) map[string]string {
	if doc == nil {
		return nil
	}
	return doc.Desired
}

func reportedOf(doc *model.Document) map[string]string {
	if doc == nil {
		return nil
	}
	return doc.Reported
}
