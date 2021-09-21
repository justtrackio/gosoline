package appctx_test

import (
	"context"
	"testing"

	"github.com/justtrackio/gosoline/pkg/appctx"
	"github.com/stretchr/testify/assert"
)

func TestMissing(t *testing.T) {
	ctx := context.Background()

	err := appctx.Set(ctx, "foo", "bar")
	assert.EqualError(t, err, "no application container found in context")

	val, err := appctx.Get(ctx, "foo")
	assert.EqualError(t, err, "no application container found in context")
	assert.Nil(t, val)
}

func TestSetAndGet(t *testing.T) {
	ctx := appctx.WithContainer(context.Background())

	err := appctx.Set(ctx, "foo", "bar")
	assert.Nil(t, err)

	val, err := appctx.Get(ctx, "foo")
	assert.NoError(t, err)
	assert.Equal(t, "bar", val)

	val, err = appctx.Get(ctx, "baz")
	assert.EqualError(t, err, "no item with key baz found")
	assert.Nil(t, val)
}

func TestGetSet(t *testing.T) {
	ctx := appctx.WithContainer(context.Background())

	val, err := appctx.GetSet(ctx, "foo", func() (interface{}, error) {
		return "bar", nil
	})
	assert.NoError(t, err)
	assert.Equal(t, "bar", val)

	val, err = appctx.GetSet(ctx, "foo", func() (interface{}, error) {
		assert.FailNow(t, "the factory should not be called a second time")
		return "bar", nil
	})
	assert.NoError(t, err)
	assert.Equal(t, "bar", val)
}
