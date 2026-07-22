package tracing_test

import (
	"context"
	"errors"
	"testing"

	"github.com/justtrackio/gosoline/pkg/appctx"
	"github.com/justtrackio/gosoline/pkg/tracing"
	"github.com/stretchr/testify/assert"
)

func TestShutdownHandler_NoProvider(t *testing.T) {
	ctx := appctx.WithContainer(context.Background())

	err := tracing.NewShutdownHandler().Shutdown(ctx)
	assert.NoError(t, err)
}

func TestShutdownHandler_CallsProvider(t *testing.T) {
	ctx := appctx.WithContainer(context.Background())

	called := false
	tracing.ProvideShutdownForTest(ctx, func(context.Context) error {
		called = true

		return nil
	})

	err := tracing.NewShutdownHandler().Shutdown(ctx)
	assert.NoError(t, err)
	assert.True(t, called)
}

func TestShutdownHandler_PropagatesError(t *testing.T) {
	ctx := appctx.WithContainer(context.Background())
	expected := errors.New("shutdown failed")

	tracing.ProvideShutdownForTest(ctx, func(context.Context) error {
		return expected
	})

	err := tracing.NewShutdownHandler().Shutdown(ctx)
	assert.ErrorIs(t, err, expected)
}

func TestShutdownHandler_NoContainer(t *testing.T) {
	err := tracing.NewShutdownHandler().Shutdown(context.Background())
	assert.NoError(t, err)
}
