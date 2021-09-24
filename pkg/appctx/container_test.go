package appctx_test

import (
	"context"
	"testing"

	"github.com/justtrackio/gosoline/pkg/appctx"
	"github.com/stretchr/testify/assert"
)

func TestProvide(t *testing.T) {
	ctx := appctx.WithContainer(context.Background())

	val, err := appctx.Provide(ctx, "foo", func() (interface{}, error) {
		return "bar", nil
	})
	assert.NoError(t, err)
	assert.Equal(t, "bar", val)

	val, err = appctx.Provide(ctx, "foo", func() (interface{}, error) {
		assert.FailNow(t, "the factory should not be called a second time")
		return "bar", nil
	})
	assert.NoError(t, err)
	assert.Equal(t, "bar", val)
}
