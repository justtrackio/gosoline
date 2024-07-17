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

	jsonType := db.NewJSON(v)

	require.Equal(t, v, jsonType.Get())

	value, err := jsonType.Value()
	require.NoError(t, err)

	b, err := json.Marshal(v)
	require.NoError(t, err)

	require.Equal(t, b, value)
	parsed := &db.JSON[T]{}
	require.NoError(t, parsed.Scan(value))

	require.Equal(t, jsonType.Get(), parsed.Get())
}

func TestJSONTypeNil(t *testing.T) {
	t.Parallel()

	type T struct {
		Foo string
	}
	var v *T

	jsonType := db.NewJSON(v)

	require.Equal(t, v, jsonType.Get())

	value, err := jsonType.Value()
	require.NoError(t, err)

	require.Nil(t, value)

	parsed := &db.JSON[*T]{}
	require.NoError(t, parsed.Scan(value))

	require.Equal(t, jsonType.Get(), parsed.Get())
}

func TestJSONTypeEmptyString(t *testing.T) {
	t.Parallel()

	var v string

	jsonType := db.NewJSON(v)

	require.Equal(t, v, jsonType.Get())

	value, err := jsonType.Value()
	require.NoError(t, err)

	require.Equal(t, "\"\"", string(value.([]byte)))

	parsed := &db.JSON[string]{}
	require.NoError(t, parsed.Scan(value))

	require.Equal(t, jsonType.Get(), parsed.Get())
}

func TestJSONTypeEmptyStringParse(t *testing.T) {
	t.Parallel()

	parsed := &db.JSON[string]{}
	require.Error(t, parsed.Scan([]byte("")))
}

func TestInvalidType(t *testing.T) {
	t.Parallel()

	parsed := &db.JSON[string]{}
	require.Error(t, parsed.Scan(""), db.ErrJSONInvalidType)
}
