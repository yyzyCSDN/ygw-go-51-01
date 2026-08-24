// Command deviceshadow runs the device shadow sync service with a browser
// console at the root path.
package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"deviceshadow/internal/conflict"
	"deviceshadow/internal/desired"
	"deviceshadow/internal/device"
	"deviceshadow/internal/model"
	"deviceshadow/internal/notify"
	"deviceshadow/internal/report"
	"deviceshadow/internal/shadow"
	"deviceshadow/internal/sync"
)

func main() {
	var (
		addr          = flag.String("addr", ":8080", "HTTP listen address")
		shards        = flag.Int("shards", 8, "device registry shard count")
		retryInterval = flag.Duration("retry-interval", 30*time.Second, "batch retry interval")
		tombstoneTTL  = flag.Duration("tombstone-ttl", 24*time.Hour, "tombstone retention")
		consolePath   = flag.String("console", "web/console.html", "console page path")
	)
	flag.Parse()

	store := shadow.NewStore()
	store.SetPolicy(conflict.PolicyForSource("console"))
	registry := device.NewRegistry(*shards)
	allocator := sync.NewAllocator()
	syncsvc := sync.NewService(store, registry)
	reportsvc := report.NewService(store, allocator)
	desiredsvc := desired.NewService(store, allocator, syncsvc)
	batchsvc := desired.NewBatchService(store, allocator, syncsvc)
	hub := notify.NewHub(store)
	store.SetOnChange(hub.Notify)

	bootstrap(registry, store, allocator, syncsvc, reportsvc)

	server := NewServer(ServerDeps{
		Store:       store,
		Registry:    registry,
		Allocator:   allocator,
		Sync:        syncsvc,
		Report:      reportsvc,
		Desired:     desiredsvc,
		Batch:       batchsvc,
		Hub:         hub,
		ConsolePath: *consolePath,
	})

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	go gcLoop(ctx, store, *tombstoneTTL)
	go retryLoop(ctx, batchsvc, *retryInterval)

	httpServer := &http.Server{
		Addr:              *addr,
		Handler:           server.Handler(),
		ReadHeaderTimeout: 5 * time.Second,
	}
	log.Printf("deviceshadow listening on %s", *addr)
	go func() {
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("http server: %v", err)
		}
	}()

	<-ctx.Done()
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.Printf("shutdown: %v", err)
	}
}

// bootstrap seeds one demo device so the console and HTTP probes have real
// state immediately after startup.
func bootstrap(registry *device.Registry, store *shadow.Store, allocator *sync.VersionAllocator, syncsvc *sync.Service, reportsvc *report.Service) {
	now := time.Now().UnixNano()
	registry.Register("demo-device-001", now)
	version := allocator.Next("demo-device-001")
	op := model.DesiredOp{
		DeviceID: "demo-device-001",
		Props:    map[string]string{"mode": "auto", "interval": "30"},
		Version:  version,
		Source:   "bootstrap",
	}
	store.ApplyDesired(op, now)
	syncsvc.Deliver("demo-device-001", op)
	reportVersion := allocator.Next("demo-device-001")
	reportOp := model.ReportOp{
		DeviceID:    "demo-device-001",
		Props:       map[string]string{"power": "on", "temp": "24"},
		BaseVersion: 0,
		Version:     reportVersion,
		ClientID:    "bootstrap",
	}
	store.ApplyReport(reportOp, now)
	reportsvc.HandleReport("demo-device-001", map[string]string{"battery": "87"}, 2, "bootstrap", now)
}
