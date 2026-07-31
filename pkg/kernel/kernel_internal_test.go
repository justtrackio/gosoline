// This test is internal because it must invoke exit and forceExit directly;
// the exported Kernel API cannot trigger forced exit while exit is blocked.
package kernel

import (
	"context"
	"testing"
	"time"

	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/stretchr/testify/assert"
)

type blockingShutdownHandler struct {
	started chan struct{}
	release chan struct{}
}

func (h *blockingShutdownHandler) Shutdown(context.Context) error {
	close(h.started)
	<-h.release

	return nil
}

func TestForceExitBypassesBlockedGracefulShutdown(t *testing.T) {
	logger := log.NewLogger()
	handler := &blockingShutdownHandler{
		started: make(chan struct{}),
		release: make(chan struct{}),
	}
	exitCodes := make(chan int, 2)
	exitDone := make(chan struct{})
	k := &kernel{
		ctx:              t.Context(),
		logger:           logger.WithChannel("kernel"),
		rootLogger:       logger,
		killTimeout:      time.Second,
		exitCode:         ExitCodeOk,
		exitHandler:      func(code int) { exitCodes <- code },
		shutdownHandlers: []ShutdownHandler{handler},
		stages:           make(stages),
	}

	go func() {
		defer close(exitDone)
		k.exit()
	}()

	select {
	case <-handler.started:
	case <-time.After(time.Second):
		t.Fatal("graceful shutdown handler did not start")
	}

	forceExitDone := make(chan struct{})
	go func() {
		defer close(forceExitDone)
		k.forceExit()
	}()

	select {
	case code := <-exitCodes:
		assert.Equal(t, ExitCodeForced, code)
	case <-time.After(time.Second):
		close(handler.release)
		t.Fatal("forced exit waited for graceful shutdown")
	}

	select {
	case <-forceExitDone:
	case <-time.After(time.Second):
		close(handler.release)
		t.Fatal("forced exit did not return")
	}

	close(handler.release)
	select {
	case <-exitDone:
	case <-time.After(time.Second):
		t.Fatal("graceful shutdown did not complete after releasing handler")
	}

	select {
	case code := <-exitCodes:
		t.Fatalf("exit handler called more than once with code %d", code)
	default:
	}
}
