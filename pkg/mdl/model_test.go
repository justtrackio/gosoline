package mdl

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"strings"
	"testing"
)

func TestModelIdFromString(t *testing.T) {
	input := "a.b.c.d"
	got, err := ModelIdFromString(input)

	assert.NoError(t, err)
	assert.Equal(t, ModelId{
		Project:     "a",
		Family:      "b",
		Application: "c",
		Name:        "d",
	}, got)
}

func FuzzModelIdFromString(f *testing.F) {
	f.Add("a", "b", "c", "d")
	f.Add("one", "two", "three", "four")
	f.Add(`本`, `当前位`, ``, `位`)

	f.Fuzz(func(t *testing.T, first string, second string, third string, fourth string) {
		if strings.Contains(first, ".") || strings.Contains(second, ".") ||
			strings.Contains(third, ".") || strings.Contains(fourth, ".") {
			return
		}

		input := fmt.Sprintf("%s.%s.%s.%s", first, second, third, fourth)
		got, err := ModelIdFromString(input)

		assert.NoError(t, err)
		assert.Equal(t, ModelId{
			Project:     first,
			Family:      second,
			Application: third,
			Name:        fourth,
		}, got)
	})
}
