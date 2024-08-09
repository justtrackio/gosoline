package refl_test

import (
	"testing"

	"github.com/justtrackio/gosoline/pkg/refl"
	"github.com/justtrackio/gosoline/pkg/test/assert"
)

func TestGetTagsNil(t *testing.T) {
	type model struct{}
	var expected []string

	tags := refl.GetTags(model{}, "json")

	assert.Equal(t, expected, tags)
}

func TestGetTagsJson(t *testing.T) {
	type model struct {
		Id     int    `json:"id"`
		Name   string `json:"name,omitempty"`
		Hidden string `json:"-"`
	}

	expected := []string{"id", "name"}

	tags1 := refl.GetTags(model{}, "json")
	assert.Equal(t, expected, tags1, "tags1 not matching")

	tags2 := refl.GetTags(&model{}, "json")
	assert.Equal(t, expected, tags2, "tags2 not matching")

	tags3 := refl.GetTags([]model{{}}, "json")
	assert.Equal(t, expected, tags3, "tags3 not matching")
}
