package db_test

import (
	"encoding/json"
	"testing"

	"github.com/justtrackio/gosoline/pkg/db"
	"github.com/stretchr/testify/require"
)

func TestJSONType(t *testing.T) {
	t.Parallel()

	v := struct{}{}

	jsonType := db.NewJSONType(v)

	require.Equal(t, v, jsonType.Get())

	value, err := jsonType.Value()
	require.NoError(t, err)

	b, err := json.Marshal(v)
	require.NoError(t, err)

	require.Equal(t, b, value)
	parsed := &db.JSONType[struct{}]{}
	require.NoError(t, parsed.Scan(value))

	require.Equal(t, jsonType.Get(), parsed.Get())
}

func TestJSONTypeAsJSONNull(t *testing.T) {
	t.Parallel()

	v := struct{}{}

	jsonType := db.NewJSONType(v)
	jsonType = jsonType.AsJSONNull()

	require.Equal(t, v, jsonType.Get())

	value, err := jsonType.Value()
	require.NoError(t, err)

	b, err := json.Marshal(v)
	require.NoError(t, err)

	require.Equal(t, b, value)
	parsed := &db.JSONType[struct{}]{}
	require.NoError(t, parsed.Scan(value))

	require.Equal(t, jsonType.Get(), parsed.Get())
}

func TestJSONTypeNilAsJSONNull(t *testing.T) {
	t.Parallel()

	var v *struct{}

	jsonType := db.NewJSONType(v)
	jsonType = jsonType.AsJSONNull()

	require.Equal(t, v, jsonType.Get())

	value, err := jsonType.Value()
	require.NoError(t, err)
	require.Equal(t, []byte("null"), value)

	b, err := json.Marshal(v)
	require.NoError(t, err)

	require.Equal(t, b, value)
	parsed := &db.JSONType[*struct{}]{}
	require.NoError(t, parsed.Scan(value))

	require.Equal(t, jsonType.Get(), parsed.Get())
}

func TestJSONTypeNil(t *testing.T) {
	t.Parallel()

	var v *struct{}

	jsonType := db.NewJSONType(v)

	require.Equal(t, v, jsonType.Get())

	value, err := jsonType.Value()
	require.NoError(t, err)

	require.Nil(t, value)

	parsed := &db.JSONType[*struct{}]{}
	require.NoError(t, parsed.Scan(value))

	require.Equal(t, jsonType.Get(), parsed.Get())
}
