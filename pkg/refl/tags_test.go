package refl_test

import (
	"testing"

	"github.com/justtrackio/gosoline/pkg/refl"
	"github.com/justtrackio/gosoline/pkg/test/assert"
)

type recursiveEmbeddedModel struct {
	*recursiveModel
}

type recursiveModel struct {
	Id int `json:"id"`
	*recursiveEmbeddedModel
}

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

func TestGetTagsEmbeddedStruct(t *testing.T) {
	type embedded struct {
		EmbeddedId   int    `json:"embeddedId"`
		EmbeddedName string `json:"embeddedName,omitempty"`
		Hidden       string `json:"-"`
	}

	type model struct {
		Id int `json:"id"`
		embedded
		Named embedded `json:"named"`
	}

	tags := refl.GetTags(model{}, "json")

	assert.Equal(t, []string{"id", "embeddedId", "embeddedName", "named"}, tags)
}

func TestGetTagsEmbeddedPointerStruct(t *testing.T) {
	type embedded struct {
		EmbeddedId int `json:"embeddedId"`
	}

	type model struct {
		Id int `json:"id"`
		*embedded
	}

	tags := refl.GetTags(model{}, "json")

	assert.Equal(t, []string{"id", "embeddedId"}, tags)
}

func TestGetTagsTaggedEmbeddedStruct(t *testing.T) {
	type embedded struct {
		EmbeddedId int `json:"embeddedId"`
	}

	type model struct {
		Id       int `json:"id"`
		embedded `json:"embedded"`
	}

	tags := refl.GetTags(model{}, "json")

	assert.Equal(t, []string{"id", "embedded"}, tags)
}

func TestGetTagsRecursiveEmbeddedPointerStruct(t *testing.T) {
	model := recursiveModel{}
	_ = model.recursiveEmbeddedModel

	tags := refl.GetTags(model, "json")

	assert.Equal(t, []string{"id"}, tags)
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
}

func TestGetTagNamesEmbeddedStruct(t *testing.T) {
	type embedded struct {
		EmbeddedId int `json:"embeddedId" db:"embedded_id"`
		Hidden     int `form:"-"`
	}

	type named struct {
		Name string `yaml:"name"`
	}

	type model struct {
		Id int `json:"id"`
		embedded
		*named
		Named named
	}

	tags := refl.GetTagNames(model{})

	assert.Equal(t, []string{"db", "form", "json", "yaml"}, tags)
}

func TestGetTagNamesTaggedEmbeddedStruct(t *testing.T) {
	type embedded struct {
		EmbeddedId int `db:"embedded_id"`
	}

	type model struct {
		Id       int `json:"id"`
		embedded `yaml:"embedded"`
	}

	tags := refl.GetTagNames(model{})

	assert.Equal(t, []string{"json", "yaml"}, tags)
}

func TestGetTagNamesRecursiveEmbeddedPointerStruct(t *testing.T) {
	model := recursiveModel{}
	_ = model.recursiveEmbeddedModel

	tags := refl.GetTagNames(model)

	assert.Equal(t, []string{"json"}, tags)
}
