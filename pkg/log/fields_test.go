package log

import (
	"fmt"
	"testing"
	"time"

	"github.com/justtrackio/gosoline/pkg/test/assert"
)

func Test_PrepareForLog_TypeError(t *testing.T) {
	toBeTested := fmt.Errorf("foo")
	prepared := prepareForLog(toBeTested)
	assert.Equal(t, "foo", prepared)
}

func Test_PrepareForLog_TypeTime(t *testing.T) {
	toBeTested := time.Now()
	prepared := prepareForLog(toBeTested)
	assert.Equal(t, toBeTested, prepared)
}

func Test_PrepareForLog_TypeMsi(t *testing.T) {
	toBeTested := map[string]any{
		"foo": "bar",
		"baz": map[string]any{
			"": "boom",
		},
	}
	expected := map[string]any{
		"foo": "bar",
		"baz": map[string]any{},
	}
	prepared := prepareForLog(toBeTested)
	assert.Equal(t, expected, prepared)
}

func Test_PrepareForLog_TypeMapSS(t *testing.T) {
	toBeTested := map[string]string{
		"foo": "bar",
		"":    "baz",
	}
	expected := map[string]any{
		"foo": "bar",
	}
	prepared := prepareForLog(toBeTested)
	assert.Equal(t, expected, prepared)
}

func Test_PrepareForLog_TypeStruct(t *testing.T) {
	type foo struct {
		Foo string
		Bar string
		_   string
	}
	toBeTested := foo{
		Foo: "bar",
		Bar: "baz",
	}
	expected := map[string]any{
		"Foo": "bar",
		"Bar": "baz",
	}
	prepared := prepareForLog(toBeTested)
	assert.Equal(t, expected, prepared)
}

func Test_PrepareForLog_NilAny(t *testing.T) {
	// An untyped nil stored in an any triggers reflect.Invalid.
	// This must not panic (regression test for zero reflect.Value).
	var v any
	prepared := prepareForLog(v)
	assert.Equal(t, nil, prepared)
}

func Test_PrepareForLog_NilPointer(t *testing.T) {
	// A typed nil pointer stored in an any has Kind == reflect.Ptr
	// with IsNil() == true.
	var p *string
	prepared := prepareForLog(p)
	assert.Equal(t, nil, prepared)
}

func Test_PrepareForLog_NonNilPointer(t *testing.T) {
	s := "hello"
	prepared := prepareForLog(&s)
	assert.Equal(t, "hello", prepared)
}

func Test_PrepareForLog_NilSlice(t *testing.T) {
	var s []string
	prepared := prepareForLog(s)
	assert.Equal(t, nil, prepared)
}

func Test_PrepareForLog_EmptySlice(t *testing.T) {
	s := []string{}
	prepared := prepareForLog(s)
	assert.Equal(t, []any{}, prepared)
}

func Test_PrepareForLog_Slice(t *testing.T) {
	s := []string{"a", "b"}
	prepared := prepareForLog(s)
	assert.Equal(t, []any{"a", "b"}, prepared)
}

func Test_PrepareForLog_Array(t *testing.T) {
	a := [2]int{1, 2}
	prepared := prepareForLog(a)
	assert.Equal(t, []any{1, 2}, prepared)
}

func Test_PrepareForLog_ScalarTypes(t *testing.T) {
	assert.Equal(t, 42, prepareForLog(42))
	assert.Equal(t, 3.14, prepareForLog(3.14))
	assert.Equal(t, true, prepareForLog(true))
	assert.Equal(t, "hello", prepareForLog("hello"))
}

func Test_PrepareForLog_NilMapValue(t *testing.T) {
	// A map with a nil value should not panic.
	m := map[string]any{
		"key": nil,
	}
	prepared := prepareForLog(m)
	expected := map[string]any{
		"key": nil,
	}
	assert.Equal(t, expected, prepared)
}

func Test_PrepareForLog_NestedNilPointer(t *testing.T) {
	// A struct containing a nil pointer field should not panic.
	type inner struct {
		Value *string
	}
	toBeTested := inner{Value: nil}
	prepared := prepareForLog(toBeTested)
	expected := map[string]any{
		"Value": nil,
	}
	assert.Equal(t, expected, prepared)
}

func Test_PrepareForLog_NestedStruct(t *testing.T) {
	type inner struct {
		X int
	}
	type outer struct {
		Inner inner
		Name  string
	}
	toBeTested := outer{
		Inner: inner{X: 5},
		Name:  "test",
	}
	prepared := prepareForLog(toBeTested)
	expected := map[string]any{
		"Inner": map[string]any{
			"X": 5,
		},
		"Name": "test",
	}
	assert.Equal(t, expected, prepared)
}

func Test_PrepareForLog_NilInterface(t *testing.T) {
	// A nil interface value stored in a map field — this is the
	// scenario that triggered the production panic via Kafka logging.
	type logger interface {
		Log()
	}
	var l logger
	prepared := prepareForLog(l)
	assert.Equal(t, nil, prepared)
}

func Test_MergeFields_NilValues(t *testing.T) {
	// mergeFields must handle nil values in both receiver and input
	// without panicking.
	receiver := map[string]any{
		"a": nil,
		"b": "value",
	}
	input := map[string]any{
		"c": nil,
	}
	result := mergeFields(receiver, input)
	expected := map[string]any{
		"a": nil,
		"b": "value",
		"c": nil,
	}
	assert.Equal(t, expected, result)
}
