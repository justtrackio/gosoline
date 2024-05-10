package appctx_test

import (
	"testing"

	"github.com/justtrackio/gosoline/pkg/appctx"
	"github.com/stretchr/testify/assert"
)

func TestProvide(t *testing.T) {
	type customType struct {
		val string
	}
	ctx := appctx.WithContainer(t.Context())

	val, err := appctx.Provide(ctx, "foo", func() (*customType, error) {
		return &customType{"bar"}, nil
	})
	assert.NoError(t, err)
	assert.Equal(t, customType{"bar"}, *val)

	val, err = appctx.Provide(ctx, "foo", func() (*customType, error) {
		assert.FailNow(t, "the factory should not be called a second time")

		return &customType{}, nil
	})
	assert.NoError(t, err)
	assert.Equal(t, customType{"bar"}, *val)
}
