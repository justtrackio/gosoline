package reqctx_test

import (
	"context"
	"testing"

	"github.com/justtrackio/gosoline/pkg/reqctx"
	"github.com/stretchr/testify/assert"
)

type typeA struct {
	value int
}

type typeB struct {
	value string
}

func TestReqCtx(t *testing.T) {
	ctx := reqctx.New(context.Background())

	a := reqctx.Get[typeA](ctx)
	b := reqctx.Get[typeB](ctx)
	assert.Nil(t, a)
	assert.Nil(t, b)

	reqctx.Set(ctx, typeA{value: 42})
	reqctx.Set(ctx, typeB{value: "abc"})

	a = reqctx.Get[typeA](ctx)
	b = reqctx.Get[typeB](ctx)
	assert.Equal(t, 42, a.value)
	assert.Equal(t, "abc", b.value)

	reqctx.Delete[typeA](ctx)
	reqctx.Set(ctx, typeB{value: "def"})

	a = reqctx.Get[typeA](ctx)
	b = reqctx.Get[typeB](ctx)
	assert.Nil(t, a)
	assert.Equal(t, "def", b.value)
}
