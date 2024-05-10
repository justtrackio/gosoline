package log_test

import (
	"testing"

	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/stretchr/testify/assert"
)

func TestInitIdempotent(t *testing.T) {
	baseCtx := t.Context()
	ctx1 := log.InitContext(baseCtx)
	assert.NotEqual(t, baseCtx, ctx1)

	ctx2 := log.InitContext(ctx1)
	assert.Equal(t, ctx1, ctx2)
}

func TestCanMutateCorrectGlobalFields(t *testing.T) {
	ctx := log.InitContext(t.Context())
	localCtx := log.AppendContextFields(ctx, map[string]any{
		"field": "value",
	})

	// mutations of local fields are only propagated to the local context
	localCtx = log.MutateContextFields(localCtx, map[string]any{
		"field": "new_value",
	})
	localFields := log.LocalOnlyContextFieldsResolver(localCtx)
	assert.Equal(t, "new_value", localFields["field"])
	// the parent context wasn't changed
	localFields = log.LocalOnlyContextFieldsResolver(ctx)
	assert.Equal(t, 0, len(localFields))

	// mutations of global fields are still propagated to the original context
	_ = log.MutateGlobalContextFields(localCtx, map[string]any{
		"other_field": "other_value",
	})
	globalFields := log.GlobalContextFieldsResolver(ctx)
	assert.Equal(t, "other_value", globalFields["other_field"])
}

func TestCanMutateCorrectLocalFields(t *testing.T) {
	ctx := log.InitContext(t.Context())
	globalCtx := log.AppendGlobalContextFields(ctx, map[string]any{
		"field": "value",
	})

	// mutations of global fields are only propagated to the global context
	globalCtx = log.MutateGlobalContextFields(globalCtx, map[string]any{
		"field": "new_value",
	})
	globalFields := log.GlobalContextFieldsResolver(globalCtx)
	assert.Equal(t, "new_value", globalFields["field"])
	// the parent context wasn't changed
	globalFields = log.GlobalContextFieldsResolver(ctx)
	assert.Equal(t, 0, len(globalFields))

	// mutations of local fields are still propagated to the original context
	_ = log.MutateContextFields(globalCtx, map[string]any{
		"other_field": "other_value",
	})
	localFields := log.LocalOnlyContextFieldsResolver(ctx)
	assert.Equal(t, "other_value", localFields["other_field"])
}

func TestMergeLoggerFieldsCorrectly(t *testing.T) {
	ctx := log.InitContext(t.Context())
	log.MutateContextFields(ctx, map[string]any{
		"field":       "local value",
		"local field": "local",
	})
	log.MutateGlobalContextFields(ctx, map[string]any{
		"field":        "global value",
		"global field": "global",
	})
	merged := log.ContextFieldsResolver(ctx)
	assert.Equal(t, map[string]any{
		"field":        "global value",
		"local field":  "local",
		"global field": "global",
	}, merged)
}
