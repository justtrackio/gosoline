package ddb_test

import (
	"github.com/applike/gosoline/pkg/ddb"
	"github.com/stretchr/testify/assert"
	"testing"
)

type TestModel struct {
	MyName         int `json:"myName"`
	MyNameWithAttr int `json:"myNameWithAttr,omitempty"`
	MyNamePlain    int `json:",omitempty"`
	MyIgnoredField int `json:"-"`
	DashField      int `json:"-,"`
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
	}, fields)
}
