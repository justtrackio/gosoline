package metric

import (
	"context"
	"errors"
	"testing"

	"github.com/justtrackio/gosoline/pkg/appctx"
	"github.com/stretchr/testify/assert"
)

func TestShutdownHandler_NoProvider(t *testing.T) {
	ctx := appctx.WithContainer(context.Background())

	err := NewShutdownHandler().Shutdown(ctx)
	assert.NoError(t, err)
}

func TestShutdownHandler_CallsProvider(t *testing.T) {
	ctx := appctx.WithContainer(context.Background())

	called := false
	_, err := appctx.Provide(ctx, metricShutdownKey{}, func() (func(context.Context) error, error) {
		return func(context.Context) error {
			called = true

			return nil
		}, nil
	})
	assert.NoError(t, err)

	err = NewShutdownHandler().Shutdown(ctx)
	assert.NoError(t, err)
	assert.True(t, called)
}

func TestShutdownHandler_PropagatesError(t *testing.T) {
	ctx := appctx.WithContainer(context.Background())
	expected := errors.New("shutdown failed")

	_, err := appctx.Provide(ctx, metricShutdownKey{}, func() (func(context.Context) error, error) {
		return func(context.Context) error {
			return expected
		}, nil
	})
	assert.NoError(t, err)

	err = NewShutdownHandler().Shutdown(ctx)
	assert.ErrorIs(t, err, expected)
}

func TestShutdownHandler_NoContainer(t *testing.T) {
	err := NewShutdownHandler().Shutdown(context.Background())
	assert.NoError(t, err)
}
