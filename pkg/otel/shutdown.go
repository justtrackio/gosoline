package otel

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"
)

// Shutdownable is implemented by OTel SDK providers (TracerProvider, MeterProvider, LoggerProvider).
type Shutdownable interface {
	Shutdown(ctx context.Context) error
}

// ShutdownSettings configures the OTel provider shutdown behavior.
type ShutdownSettings struct {
	Timeout time.Duration `cfg:"timeout" default:"10s"`
}

type shutdownEntry struct {
	priority int
	resource Shutdownable
}

const (
	PriorityMetrics = 0
	PriorityTraces  = 10
	PriorityLogs    = 20
)

var (
	shutdownMu      sync.Mutex
	shutdownEntries []shutdownEntry
)

// Register adds a Shutdownable resource to the global shutdown registry.
// Lower priority values are shut down first.
func Register(priority int, resource Shutdownable) {
	shutdownMu.Lock()
	defer shutdownMu.Unlock()

	shutdownEntries = append(shutdownEntries, shutdownEntry{
		priority: priority,
		resource: resource,
	})
}

// ShutdownAll shuts down all registered resources in priority order (lowest first).
// It returns a combined error of all shutdown failures.
func ShutdownAll(ctx context.Context) error {
	shutdownMu.Lock()
	entries := make([]shutdownEntry, len(shutdownEntries))
	copy(entries, shutdownEntries)
	shutdownMu.Unlock()

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].priority < entries[j].priority
	})

	var errs []error
	for _, entry := range entries {
		if err := entry.resource.Shutdown(ctx); err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) == 0 {
		return nil
	}

	return fmt.Errorf("otel shutdown errors: %v", errs)
}

// ResetRegistry clears the shutdown registry. Intended for testing only.
func ResetRegistry() {
	shutdownMu.Lock()
	defer shutdownMu.Unlock()

	shutdownEntries = nil
}
