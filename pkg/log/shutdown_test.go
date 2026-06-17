package log_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/stretchr/testify/assert"
)

func TestShutdownHandler_Empty(t *testing.T) {
	log.ResetShutdownRegistry()
	t.Cleanup(log.ResetShutdownRegistry)

	err := log.NewShutdownHandler().Shutdown(context.Background())
	assert.NoError(t, err)
}

func TestShutdownHandler_RunsInRegistrationOrder(t *testing.T) {
	log.ResetShutdownRegistry()
	t.Cleanup(log.ResetShutdownRegistry)

	var order []string
	log.RegisterShutdown("first", func(_ context.Context) error {
		order = append(order, "first")

		return nil
	})
	log.RegisterShutdown("second", func(_ context.Context) error {
		order = append(order, "second")

		return nil
	})

	err := log.NewShutdownHandler().Shutdown(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, []string{"first", "second"}, order)
}

func TestShutdownHandler_AggregatesErrorsAndContinues(t *testing.T) {
	log.ResetShutdownRegistry()
	t.Cleanup(log.ResetShutdownRegistry)

	var called []string
	log.RegisterShutdown("failing", func(_ context.Context) error {
		called = append(called, "failing")

		return fmt.Errorf("boom")
	})
	log.RegisterShutdown("healthy", func(_ context.Context) error {
		called = append(called, "healthy")

		return nil
	})
	log.RegisterShutdown("failing2", func(_ context.Context) error {
		called = append(called, "failing2")

		return fmt.Errorf("bang")
	})

	err := log.NewShutdownHandler().Shutdown(context.Background())

	assert.Error(t, err)
	assert.ErrorContains(t, err, "failing: boom")
	assert.ErrorContains(t, err, "failing2: bang")
	assert.Equal(t, []string{"failing", "healthy", "failing2"}, called, "all functions run despite failures")
}

func TestShutdownHandler_PassesContext(t *testing.T) {
	log.ResetShutdownRegistry()
	t.Cleanup(log.ResetShutdownRegistry)

	type ctxKey string
	key := ctxKey("marker")
	ctx := context.WithValue(context.Background(), key, "value")

	var got any
	log.RegisterShutdown("capture", func(ctx context.Context) error {
		got = ctx.Value(key)

		return nil
	})

	err := log.NewShutdownHandler().Shutdown(ctx)
	assert.NoError(t, err)
	assert.Equal(t, "value", got)
}
