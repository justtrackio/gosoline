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

func TestGetTagsNested(t *testing.T) {
	type DeeplyNested struct {
		Value int `json:"value"`
	}

	type Nested struct {
		Name string `json:"name"`
		DeeplyNested
	}

	type model struct {
		Id int `json:"id"`
		Nested
	}

	expected := []string{"id", "name", "value"}

	tags1 := refl.GetTags(model{}, "json")
	assert.Equal(t, expected, tags1, "tags1 not matching")

	tags2 := refl.GetTags(&model{}, "json")
	assert.Equal(t, expected, tags2, "tags2 not matching")

	tags3 := refl.GetTags([]model{{}}, "json")
	assert.Equal(t, expected, tags3, "tags3 not matching")
}

func TestGetTagNames(t *testing.T) {
	// base struct with multiple tags on a field
	type model struct {
		Id     int    `json:"id"  yaml:"id" db:"id"`
		Name   string `db:"name,omitempty"`
		Hidden string `form:"-"`
	}
	tags := refl.GetTagNames(model{})
	assert.Equal(t, []string{"db", "form", "json", "yaml"}, tags, "basic model")

	// pointer to struct
	ptags := refl.GetTagNames(&model{})
	assert.Equal(t, []string{"db", "form", "json", "yaml"}, ptags, "pointer model")

	// slice of struct
	stags := refl.GetTagNames([]model{{}})
	assert.Equal(t, []string{"db", "form", "json", "yaml"}, stags, "slice model")

	// struct with duplicated tag keys across fields
	type modelDup struct {
		A int `json:"a"`
		B int `json:"b" db:"b"`
		C int `xml:"c"`
		D int `db:"d" xml:"d"`
	}
	tagsDup := refl.GetTagNames(modelDup{})
	assert.Equal(t, []string{"db", "json", "xml"}, tagsDup, "duplicate tag keys aggregated once")

	// struct with empty tag values
	type modelEmpty struct {
		A int `yaml:""`
		B int `toml:""`
	}
	tagsEmpty := refl.GetTagNames(modelEmpty{})
	assert.Equal(t, []string{"toml", "yaml"}, tagsEmpty, "empty values still count tag names")

	// struct without tags
	type modelNone struct {
		A int
		B string
	}
	tagsNone := refl.GetTagNames(modelNone{})
	assert.Equal(t, []string{}, tagsNone, "no tags should return empty slice")

	// pointer nil (should yield empty slice)
	var ptr *modelNone
	ptrTags := refl.GetTagNames(ptr)
	assert.Equal(t, []string{}, ptrTags, "nil pointer should return empty slice")

	// struct with nested tag values
	type modelNested struct {
		model
		modelDup
		modelEmpty
		X int `validate:"x"`
	}
	tagsNested := refl.GetTagNames(modelNested{})
	assert.Equal(t, []string{"db", "form", "json", "toml", "validate", "xml", "yaml"}, tagsNested, "nested structs are correctly handled")
}

// TODO: nested struct pointers
// TODO: nested struct pointer loops?!
