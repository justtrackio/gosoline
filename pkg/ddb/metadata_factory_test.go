package ddb_test

import (
	"testing"

	"github.com/justtrackio/gosoline/pkg/ddb"
	"github.com/stretchr/testify/assert"
)

type TestModel struct {
	MyName         int    `json:"myName" ddb:"key=hash"`
	MyNameWithAttr string `json:"myNameWithAttr,omitempty" ddb:"global=hash"`
	MyNamePlain    string `json:",omitempty" ddb:"key=range"`
	MyIgnoredField int64  `json:"-"`
	DashField      int    `json:"-," ddb:"global=range"`
	DefaultField   uint   `ddb:"global=range"`
}

type TestModelEmptyDDB struct {
	Field int `ddb:""`
}

type TestModelNoKV struct {
	Field int `ddb:"not a value"`
}

type TestModelEmptyJSON struct {
	Field int `json:"" ddb:"key=hash"`
}

func TestReadAttributes(t *testing.T) {
	attributes, err := ddb.ReadAttributes(TestModel{})
	assert.NoError(t, err)
	assert.Equal(t, ddb.Attributes{
		"myName": &ddb.Attribute{
			FieldName:     "MyName",
			AttributeName: "myName",
			Tags: map[string]string{
				"key": "hash",
			},
			Type: "N",
		},
		"myNameWithAttr": &ddb.Attribute{
			FieldName:     "MyNameWithAttr",
			AttributeName: "myNameWithAttr",
			Tags: map[string]string{
				"global": "hash",
			},
			Type: "S",
		},
		"MyNamePlain": &ddb.Attribute{
			FieldName:     "MyNamePlain",
			AttributeName: "MyNamePlain",
			Tags: map[string]string{
				"key": "range",
			},
			Type: "S",
		},
		"-": &ddb.Attribute{
			FieldName:     "DashField",
			AttributeName: "-",
			Tags: map[string]string{
				"global": "range",
			},
			Type: "N",
		},
		"DefaultField": &ddb.Attribute{
			FieldName:     "DefaultField",
			AttributeName: "DefaultField",
			Tags: map[string]string{
				"global": "range",
			},
			Type: "N",
		},
	}, attributes)
	_, err = ddb.ReadAttributes(TestModelEmptyDDB{})
	assert.Error(t, err)
	_, err = ddb.ReadAttributes(TestModelEmptyJSON{})
	assert.Error(t, err)
	_, err = ddb.ReadAttributes(TestModelNoKV{})
	assert.Error(t, err)
}

func TestMetadataReadFields(t *testing.T) {
	fields, err := ddb.MetadataReadFields(TestModel{})
	assert.NoError(t, err)
	assert.Equal(t, []string{
		"myName",
		"myNameWithAttr",
		"MyNamePlain",
		// MyIgnoredField is... ignored, so there is no entry for it
		"-",
		"DefaultField",
	}, fields)
	_, err = ddb.MetadataReadFields(TestModelEmptyJSON{})
	assert.Error(t, err)
}
