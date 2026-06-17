package tracing_test

import (
	"context"
	"errors"
	"testing"

	"github.com/justtrackio/gosoline/pkg/tracing"
	"github.com/stretchr/testify/assert"
)

func TestShutdownHandler_Empty(t *testing.T) {
	tracing.ResetShutdownRegistry()
	t.Cleanup(tracing.ResetShutdownRegistry)

	assert.NoError(t, tracing.NewShutdownHandler().Shutdown(t.Context()))
}

func TestShutdownHandler_RunsInRegistrationOrderAndContinues(t *testing.T) {
	tracing.ResetShutdownRegistry()
	t.Cleanup(tracing.ResetShutdownRegistry)

	firstErr := errors.New("first boom")
	secondErr := errors.New("second boom")
	var calls []string
	tracing.RegisterShutdown("first-failure", func(context.Context) error {
		calls = append(calls, "first-failure")

		return firstErr
	})
	tracing.RegisterShutdown("successful", func(context.Context) error {
		calls = append(calls, "successful")

		return nil
	})
	tracing.RegisterShutdown("second-failure", func(context.Context) error {
		calls = append(calls, "second-failure")

		return secondErr
	})

	err := tracing.NewShutdownHandler().Shutdown(t.Context())
	assert.ErrorContains(t, err, "first-failure: first boom")
	assert.ErrorContains(t, err, "second-failure: second boom")
	assert.ErrorIs(t, err, firstErr)
	assert.ErrorIs(t, err, secondErr)
	assert.Equal(t, []string{"first-failure", "successful", "second-failure"}, calls)
}
