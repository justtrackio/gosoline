package db_test

import (
	"encoding/json"
	"testing"

	"github.com/justtrackio/gosoline/pkg/db"
	"github.com/stretchr/testify/require"
)

func TestJSONType(t *testing.T) {
	t.Parallel()

	type T struct {
		Foo string
	}
	v := T{
		Foo: "bar",
	}

	jsonType := db.NewJSONType(v)

	require.Equal(t, v, jsonType.Get())

	value, err := jsonType.Value()
	require.NoError(t, err)

	b, err := json.Marshal(v)
	require.NoError(t, err)

	require.Equal(t, b, value)
	parsed := &db.JSONType[T]{}
	require.NoError(t, parsed.Scan(value))

	require.Equal(t, jsonType.Get(), parsed.Get())
}

func TestJSONTypeAsJSONNull(t *testing.T) {
	t.Parallel()

	type T struct {
		Foo string
	}
	v := T{
		Foo: "bar",
	}

	jsonType := db.NewJSONType(v)
	jsonType = jsonType.AsJSONNull()

	require.Equal(t, v, jsonType.Get())

	value, err := jsonType.Value()
	require.NoError(t, err)

	b, err := json.Marshal(v)
	require.NoError(t, err)

	require.Equal(t, b, value)
	parsed := &db.JSONType[T]{}
	require.NoError(t, parsed.Scan(value))

	require.Equal(t, jsonType.Get(), parsed.Get())
}

func TestJSONTypeNilAsJSONNull(t *testing.T) {
	t.Parallel()

	type T struct {
		Foo string
	}
	var v *T

	jsonType := db.NewJSONType(v)
	jsonType = jsonType.AsJSONNull()

	require.Equal(t, v, jsonType.Get())

	value, err := jsonType.Value()
	require.NoError(t, err)
	require.Equal(t, []byte("null"), value)

	b, err := json.Marshal(v)
	require.NoError(t, err)

	require.Equal(t, b, value)
	parsed := &db.JSONType[*T]{}
	require.NoError(t, parsed.Scan(value))

	require.Equal(t, jsonType.Get(), parsed.Get())
}

func TestJSONTypeNil(t *testing.T) {
	t.Parallel()

	type T struct {
		Foo string
	}
	var v *T

	jsonType := db.NewJSONType(v)

	require.Equal(t, v, jsonType.Get())

	value, err := jsonType.Value()
	require.NoError(t, err)

	require.Nil(t, value)

	parsed := &db.JSONType[*T]{}
	require.NoError(t, parsed.Scan(value))

	require.Equal(t, jsonType.Get(), parsed.Get())
}

func TestJSONTypeEmptyString(t *testing.T) {
	t.Parallel()

	var v string

	jsonType := db.NewJSONType(v)

	require.Equal(t, v, jsonType.Get())

	value, err := jsonType.Value()
	require.NoError(t, err)

	require.Equal(t, "\"\"", string(value.([]byte)))

	parsed := &db.JSONType[string]{}
	require.NoError(t, parsed.Scan(value))

	require.Equal(t, jsonType.Get(), parsed.Get())
}

func TestJSONTypeEmptyStringParse(t *testing.T) {
	t.Parallel()

	parsed := &db.JSONType[string]{}
	require.Error(t, parsed.Scan([]byte("")))
}

func TestInvalidDriverType(t *testing.T) {
	t.Parallel()

	parsed := &db.JSONType[string]{}
	require.Error(t, parsed.Scan(""), db.ErrJSONInvalidDriverType)
}
