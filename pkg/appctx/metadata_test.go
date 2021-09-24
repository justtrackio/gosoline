package appctx_test

import (
	"testing"

	"github.com/justtrackio/gosoline/pkg/appctx"
	"github.com/stretchr/testify/assert"
)

func TestMetadataAppend(t *testing.T) {
	metadata := appctx.NewMetadata()

	err := metadata.Append("key", "foo")
	assert.NoError(t, err)

	act, err := metadata.Get("key").Slice()
	assert.NoError(t, err)
	assert.Equal(t, []interface{}{"foo"}, act)

	// append duplicate string
	err = metadata.Append("key", "foo")
	assert.NoError(t, err)

	act, err = metadata.Get("key").Slice()
	assert.NoError(t, err)
	assert.Equal(t, []interface{}{"foo"}, act)

	// append new string
	err = metadata.Append("key", "bar")
	assert.NoError(t, err)

	act, err = metadata.Get("key").Slice()
	assert.NoError(t, err)
	assert.Equal(t, []interface{}{"foo", "bar"}, act)

	// test with struct types
	type someStructValue struct {
		I int
		S string
	}

	err = metadata.Append("structKey", someStructValue{1, "foo"})
	assert.NoError(t, err)

	act, err = metadata.Get("structKey").Slice()
	assert.NoError(t, err)
	assert.Equal(t, []interface{}{someStructValue{1, "foo"}}, act)

	// append duplicate someStructValue
	err = metadata.Append("structKey", someStructValue{1, "foo"})
	assert.NoError(t, err)

	act, err = metadata.Get("structKey").Slice()
	assert.NoError(t, err)
	assert.Equal(t, []interface{}{someStructValue{1, "foo"}}, act)

	// append new someStructValue
	err = metadata.Append("structKey", someStructValue{1, "bar"})
	assert.NoError(t, err)

	act, err = metadata.Get("structKey").Slice()
	assert.NoError(t, err)
	assert.Equal(t, []interface{}{someStructValue{1, "foo"}, someStructValue{1, "bar"}}, act)
}
