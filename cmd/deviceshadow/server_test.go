package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"deviceshadow/internal/conflict"
	"deviceshadow/internal/desired"
	"deviceshadow/internal/device"
	"deviceshadow/internal/notify"
	"deviceshadow/internal/report"
	"deviceshadow/internal/shadow"
	"deviceshadow/internal/sync"
)

func TestHealthEndpoint(t *testing.T) {
	server := newTestServer(t)
	request := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	recorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("health endpoint returned %d", recorder.Code)
	}
	var body map[string]string
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid health body: %v", err)
	}
	if body["status"] != "ok" {
		t.Fatalf("unexpected health body: %v", body)
	}
}

func TestRegisterAndReadShadow(t *testing.T) {
	server := newTestServer(t)
	register := httptest.NewRequest(http.MethodPost, "/api/v1/devices", strings.NewReader(`{"device_id":"probe-1"}`))
	register.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(recorder, register)
	if recorder.Code != http.StatusCreated {
		t.Fatalf("register returned %d: %s", recorder.Code, recorder.Body.String())
	}

	reportBody := strings.NewReader(`{"props":{"temp":"26"},"version":0,"client_id":"probe"}`)
	reportReq := httptest.NewRequest(http.MethodPost, "/api/v1/devices/probe-1/report", reportBody)
	reportReq.Header.Set("Content-Type", "application/json")
	reportRecorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(reportRecorder, reportReq)
	if reportRecorder.Code != http.StatusOK {
		t.Fatalf("report returned %d: %s", reportRecorder.Code, reportRecorder.Body.String())
	}

	get := httptest.NewRequest(http.MethodGet, "/api/v1/devices/probe-1/shadow", nil)
	getRecorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(getRecorder, get)
	if getRecorder.Code != http.StatusOK {
		t.Fatalf("get shadow returned %d", getRecorder.Code)
	}
	if !strings.Contains(getRecorder.Body.String(), "26") {
		t.Fatalf("reported value missing from shadow: %s", getRecorder.Body.String())
	}
}

func newTestServer(t *testing.T) *Server {
	t.Helper()
	store := shadow.NewStore()
	store.SetPolicy(conflict.DefaultPolicy())
	registry := device.NewRegistry(4)
	allocator := sync.NewAllocator()
	syncsvc := sync.NewService(store, registry)
	reportsvc := report.NewService(store, allocator)
	desiredsvc := desired.NewService(store, allocator, syncsvc)
	batchsvc := desired.NewBatchService(store, allocator, syncsvc)
	hub := notify.NewHub(store)
	store.SetOnChange(hub.Notify)
	return NewServer(ServerDeps{
		Store:       store,
		Registry:    registry,
		Allocator:   allocator,
		Sync:        syncsvc,
		Report:      reportsvc,
		Desired:     desiredsvc,
		Batch:       batchsvc,
		Hub:         hub,
		ConsolePath: "web/console.html",
	})
}
