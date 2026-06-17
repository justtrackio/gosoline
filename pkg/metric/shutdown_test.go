package metric

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestShutdownHandler_Empty(t *testing.T) {
	ResetShutdownRegistry()
	t.Cleanup(ResetShutdownRegistry)

	assert.NoError(t, NewShutdownHandler().Shutdown(t.Context()))
}

func TestShutdownHandler_RunsInRegistrationOrderAndContinues(t *testing.T) {
	ResetShutdownRegistry()
	t.Cleanup(ResetShutdownRegistry)

	firstErr := errors.New("first boom")
	secondErr := errors.New("second boom")
	var calls []string
	RegisterShutdown("first-failure", func(context.Context) error {
		calls = append(calls, "first-failure")

		return firstErr
	})
	RegisterShutdown("successful", func(context.Context) error {
		calls = append(calls, "successful")

		return nil
	})
	RegisterShutdown("second-failure", func(context.Context) error {
		calls = append(calls, "second-failure")

		return secondErr
	})

	err := NewShutdownHandler().Shutdown(context.Background())
	assert.ErrorContains(t, err, "first-failure: first boom")
	assert.ErrorContains(t, err, "second-failure: second boom")
	assert.ErrorIs(t, err, firstErr)
	assert.ErrorIs(t, err, secondErr)
	assert.Equal(t, []string{"first-failure", "successful", "second-failure"}, calls)
}
