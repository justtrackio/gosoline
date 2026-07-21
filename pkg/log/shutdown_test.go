package log_test

import (
	"context"
	"errors"
	"testing"

	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/stretchr/testify/assert"
)

func TestShutdownHandler_NoProvider(t *testing.T) {
	ctx := log.WithShutdownContainer(context.Background())

	err := log.NewShutdownHandler().Shutdown(ctx)
	assert.NoError(t, err)
}

func TestShutdownHandler_CallsProvider(t *testing.T) {
	called := false
	ctx := log.ProvideShutdownForTest(context.Background(), func(context.Context) error {
		called = true

		return nil
	})

	err := log.NewShutdownHandler().Shutdown(ctx)
	assert.NoError(t, err)
	assert.True(t, called)
}

func TestShutdownHandler_PropagatesError(t *testing.T) {
	expected := errors.New("shutdown failed")

	ctx := log.ProvideShutdownForTest(context.Background(), func(context.Context) error {
		return expected
	})

	err := log.NewShutdownHandler().Shutdown(ctx)
	assert.ErrorIs(t, err, expected)
}

func TestShutdownHandler_NoContainer(t *testing.T) {
	err := log.NewShutdownHandler().Shutdown(context.Background())
	assert.NoError(t, err)
}
