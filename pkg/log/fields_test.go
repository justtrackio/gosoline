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
	toBeTested := map[string]interface{}{
		"foo": "bar",
		"baz": map[string]interface{}{
			"": "boom",
		},
	}
	expected := map[string]interface{}{
		"foo": "bar",
		"baz": map[string]interface{}{},
	}
	prepared := prepareForLog(toBeTested)
	assert.Equal(t, expected, prepared)
}

func Test_PrepareForLog_TypeMapSS(t *testing.T) {
	toBeTested := map[string]string{
		"foo": "bar",
		"":    "baz",
	}
	expected := map[string]interface{}{
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
	expected := map[string]interface{}{
		"Foo": "bar",
		"Bar": "baz",
	}
	prepared := prepareForLog(toBeTested)
	assert.Equal(t, expected, prepared)
}
