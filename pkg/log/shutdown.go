package log

import (
	"context"
	"errors"
	"fmt"
	"sync"
)

// ShutdownHandler drains the registered log backend shutdown functions. It is implemented
// by the value returned from NewShutdownHandler and consumed by the kernel, which runs it
// after emitting the final exit-code log line.
type ShutdownHandler interface {
	Shutdown(ctx context.Context) error
}

type shutdownEntry struct {
	name string
	fn   func(ctx context.Context) error
}

var (
	shutdownMu      sync.Mutex
	shutdownEntries []shutdownEntry
)

// RegisterShutdown registers a log backend shutdown/flush function to be executed when the
// application exits, after the final log line has been emitted. Functions run in
// registration order. It is safe to call from init/bootstrap code before the kernel exists.
func RegisterShutdown(name string, fn func(ctx context.Context) error) {
	shutdownMu.Lock()
	defer shutdownMu.Unlock()

	shutdownEntries = append(shutdownEntries, shutdownEntry{
		name: name,
		fn:   fn,
	})
}

// ResetShutdownRegistry clears the shutdown registry. Intended for testing only.
func ResetShutdownRegistry() {
	shutdownMu.Lock()
	defer shutdownMu.Unlock()

	shutdownEntries = nil
}

// NewShutdownHandler returns a ShutdownHandler backed by the package-level registry.
func NewShutdownHandler() ShutdownHandler {
	return registryShutdownHandler{}
}

type registryShutdownHandler struct{}

// Shutdown runs all registered shutdown functions in registration order. It aggregates
// errors and continues on failure so that a single failing backend does not prevent the
// others from flushing.
func (registryShutdownHandler) Shutdown(ctx context.Context) error {
	shutdownMu.Lock()
	entries := make([]shutdownEntry, len(shutdownEntries))
	copy(entries, shutdownEntries)
	shutdownMu.Unlock()

	var errs []error
	for _, entry := range entries {
		if err := entry.fn(ctx); err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", entry.name, err))
		}
	}

	return errors.Join(errs...)
}
