package tracing

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/justtrackio/gosoline/pkg/kernel"
)

type shutdownEntry struct {
	name string
	fn   func(context.Context) error
}

var (
	shutdownMu      sync.Mutex
	shutdownEntries []shutdownEntry
)

// RegisterShutdown registers a tracing provider shutdown function to be executed when the
// application exits. Functions run in registration order.
func RegisterShutdown(name string, fn func(context.Context) error) {
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
func NewShutdownHandler() kernel.ShutdownHandler {
	return registryShutdownHandler{}
}

type registryShutdownHandler struct{}

var _ kernel.ShutdownHandler = registryShutdownHandler{}

// Shutdown runs all registered tracing provider shutdown functions in registration order.
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
