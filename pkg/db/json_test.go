package db_test

import (
	"encoding/json"
	"testing"

	"github.com/justtrackio/gosoline/pkg/db"
	"github.com/stretchr/testify/require"
)

type (
	JSONTestCase[BehaviourFrom, BehaviourTo db.Nullable | db.NonNullable] struct {
		description
		fromBehaviour BehaviourFrom
		toBehaviour   BehaviourTo
	}

	description string

	Runnable interface {
		Run(t *testing.T)
	}
	Describable interface {
		Desc() string
	}

	StandardTestCase[BehaviourFrom, BehaviourTo db.Nullable | db.NonNullable]    JSONTestCase[BehaviourFrom, BehaviourTo]
	NilTestCase[BehaviourFrom, BehaviourTo db.Nullable | db.NonNullable]         JSONTestCase[BehaviourFrom, BehaviourTo]
	EmptyStringTestCase[BehaviourFrom, BehaviourTo db.Nullable | db.NonNullable] JSONTestCase[BehaviourFrom, BehaviourTo]
	NilMapTestCase[BehaviourFrom, BehaviourTo db.Nullable | db.NonNullable]      JSONTestCase[BehaviourFrom, BehaviourTo]
	NilSliceTestCase[BehaviourFrom, BehaviourTo db.Nullable | db.NonNullable]    JSONTestCase[BehaviourFrom, BehaviourTo]
)

func (testCase StandardTestCase[BehaviourFrom, BehaviourTo]) Run(t *testing.T) {
	t.Parallel()

	type T struct {
		Foo string
	}
	v := T{
		Foo: "bar",
	}

	jsonType := db.NewJSON(v, testCase.fromBehaviour)

	require.Equal(t, v, jsonType.Get())

	value, err := jsonType.Value()
	require.NoError(t, err)

	b, err := json.Marshal(v)
	require.NoError(t, err)

	require.Equal(t, b, value)
	parsed := &db.JSON[T, BehaviourTo]{}
	require.NoError(t, parsed.Scan(value))

	require.Equal(t, jsonType.Get(), parsed.Get())
}

func (testCase NilTestCase[BehaviourFrom, BehaviourTo]) Run(t *testing.T) {
	t.Parallel()

	type T struct {
		Foo string
	}
	var v *T

	jsonType := db.NewJSON(v, testCase.fromBehaviour)

	require.Equal(t, v, jsonType.Get())

	value, err := jsonType.Value()
	require.NoError(t, err)

	if db.IsNullable(testCase.fromBehaviour).IsNullable() {
		require.Nil(t, value)
	} else {
		require.Equal(t, []byte("null"), value)
	}

	parsed := &db.JSON[*T, BehaviourTo]{}
	require.NoError(t, parsed.Scan(value))

	require.Equal(t, jsonType.Get(), parsed.Get())
}

func (testCase EmptyStringTestCase[BehaviourFrom, BehaviourTo]) Run(t *testing.T) {
	t.Parallel()

	var v string

	jsonType := db.NewJSON(v, testCase.fromBehaviour)

	require.Equal(t, v, jsonType.Get())

	value, err := jsonType.Value()
	require.NoError(t, err)

	require.Equal(t, "\"\"", string(value.([]byte)))

	parsed := &db.JSON[string, BehaviourTo]{}
	require.NoError(t, parsed.Scan(value))

	require.Equal(t, jsonType.Get(), parsed.Get())
}

func (testCase NilSliceTestCase[BehaviourFrom, BehaviourTo]) Run(t *testing.T) {
	t.Parallel()

	type T struct {
		Foo string
	}
	var v []T

	jsonType := db.NewJSON(v, testCase.fromBehaviour)

	require.Equal(t, v, jsonType.Get())

	value, err := jsonType.Value()
	require.NoError(t, err)

	if db.IsNullable(testCase.fromBehaviour).IsNullable() {
		require.Nil(t, value)
	} else {
		require.Equal(t, []byte("null"), value)
	}

	parsed := &db.JSON[*T, BehaviourTo]{}
	require.NoError(t, parsed.Scan(value))

	require.Equal(t, jsonType.Get(), parsed.Get())
}

func (testCase NilMapTestCase[BehaviourFrom, BehaviourTo]) Run(t *testing.T) {
	t.Parallel()

	type T struct {
		Foo string
	}
	var v map[T]T

	jsonType := db.NewJSON(v, testCase.fromBehaviour)

	require.Equal(t, v, jsonType.Get())

	value, err := jsonType.Value()
	require.NoError(t, err)

	if db.IsNullable(testCase.fromBehaviour).IsNullable() {
		require.Nil(t, value)
	} else {
		require.Equal(t, []byte("null"), value)
	}

	parsed := &db.JSON[*T, BehaviourTo]{}
	require.NoError(t, parsed.Scan(value))

	require.Equal(t, jsonType.Get(), parsed.Get())
}

func (d description) Desc() string {
	return string(d)
}

func TestJSON(t *testing.T) {
	testCases := []interface {
		Runnable
		Describable
	}{
		StandardTestCase[db.NonNullable, db.NonNullable]{
			description: "NonNullable To NonNullable",
		},
		StandardTestCase[db.Nullable, db.NonNullable]{
			description: "Nullable To NonNullable",
		},
		StandardTestCase[db.NonNullable, db.Nullable]{
			description: "NonNullable To Nullable",
		},
		StandardTestCase[db.Nullable, db.Nullable]{
			description: "Nullable To Nullable",
		},
	}
	for _, tC := range testCases {
		t.Run(tC.Desc(), tC.Run)
	}
}

func TestJSONNil(t *testing.T) {
	testCases := []interface {
		Runnable
		Describable
	}{
		NilTestCase[db.NonNullable, db.NonNullable]{
			description: "NonNullable To NonNullable",
		},
		NilTestCase[db.Nullable, db.NonNullable]{
			description: "Nullable To NonNullable",
		},
		NilTestCase[db.NonNullable, db.Nullable]{
			description: "NonNullable To Nullable",
		},
		NilTestCase[db.Nullable, db.Nullable]{
			description: "Nullable To Nullable",
		},
	}
	for _, tC := range testCases {
		t.Run(tC.Desc(), tC.Run)
	}
}

func TestJSONEmptyString(t *testing.T) {
	testCases := []interface {
		Runnable
		Describable
	}{
		EmptyStringTestCase[db.NonNullable, db.NonNullable]{
			description: "NonNullable To NonNullable",
		},
		EmptyStringTestCase[db.Nullable, db.NonNullable]{
			description: "Nullable To NonNullable",
		},
		EmptyStringTestCase[db.NonNullable, db.Nullable]{
			description: "NonNullable To Nullable",
		},
		EmptyStringTestCase[db.Nullable, db.Nullable]{
			description: "Nullable To Nullable",
		},
	}
	for _, tC := range testCases {
		t.Run(tC.Desc(), tC.Run)
	}
}

func TestJSONNilMap(t *testing.T) {
	testCases := []interface {
		Runnable
		Describable
	}{
		EmptyStringTestCase[db.NonNullable, db.NonNullable]{
			description: "NonNullable To NonNullable",
		},
		EmptyStringTestCase[db.Nullable, db.NonNullable]{
			description: "Nullable To NonNullable",
		},
		EmptyStringTestCase[db.NonNullable, db.Nullable]{
			description: "NonNullable To Nullable",
		},
		EmptyStringTestCase[db.Nullable, db.Nullable]{
			description: "Nullable To Nullable",
		},
	}
	for _, tC := range testCases {
		t.Run(tC.Desc(), tC.Run)
	}
}

func TestJSONNilSlice(t *testing.T) {
	testCases := []interface {
		Runnable
		Describable
	}{
		EmptyStringTestCase[db.NonNullable, db.NonNullable]{
			description: "NonNullable To NonNullable",
		},
		EmptyStringTestCase[db.Nullable, db.NonNullable]{
			description: "Nullable To NonNullable",
		},
		EmptyStringTestCase[db.NonNullable, db.Nullable]{
			description: "NonNullable To Nullable",
		},
		EmptyStringTestCase[db.Nullable, db.Nullable]{
			description: "Nullable To Nullable",
		},
	}
	for _, tC := range testCases {
		t.Run(tC.Desc(), tC.Run)
	}
}

func TestJSONTypeEmptyStringParseNonNullable(t *testing.T) {
	t.Parallel()

	parsed := &db.JSON[string, db.NonNullable]{}
	require.Error(t, parsed.Scan([]byte("")))
}

func TestJSONTypeEmptyStringParseNullable(t *testing.T) {
	t.Parallel()

	parsed := &db.JSON[string, db.Nullable]{}
	require.Error(t, parsed.Scan([]byte("")))
}

func TestInvalidType(t *testing.T) {
	t.Parallel()

	parsed := &db.JSON[string, db.Nullable]{}
	require.Error(t, parsed.Scan(""), db.ErrJSONInvalidType)
}
