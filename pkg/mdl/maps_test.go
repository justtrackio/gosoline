package mdl_test

import (
	"github.com/applike/gosoline/pkg/encoding/json"
	"github.com/applike/gosoline/pkg/encoding/yaml"
	"github.com/applike/gosoline/pkg/mdl"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"testing"
)

var ok bool
var str string
var vsm mdl.ValueStringMap
var vim mdl.ValueInterfaceMap

func readFile(t *testing.T, path string) []byte {
	data, err := ioutil.ReadFile(path)

	if err != nil {
		assert.Fail(t, "could not read file", path)
	}

	return data
}

type MyStructValue struct {
	Foo      string             `json:"foo"`
	Map      mdl.ValueStringMap `json:"map"`
	NotExist mdl.ValueStringMap `json:"notExist"`
}

func TestValueMap_Json_Struct(t *testing.T) {
	root := MyStructValue{}
	data := readFile(t, "testdata/struct.json")
	err := json.Unmarshal(data, &root)

	if err != nil {
		assert.Fail(t, "could not unmarshal json")
	}

	assert.Equal(t, "bar", root.Foo)
	assert.NotNil(t, root.Map)

	vsm, ok := root.NotExist.GetMap("mh")
	assert.Nil(t, vsm)
	assert.False(t, ok)
}

func TestValueMap_Json_Raw(t *testing.T) {
	vm := mdl.ValueStringMap{}
	data := readFile(t, "testdata/raw.json")
	err := json.Unmarshal(data, &vm)

	if err != nil {
		assert.Fail(t, "could not unmarshal json")
	}

	str, ok = vm.GetString("a")
	assert.True(t, ok)
	assert.Equal(t, "b", str)

	vsm, ok = vm.GetMap("map")
	assert.True(t, ok)
	assert.IsType(t, mdl.ValueStringMap{}, vsm)
}

func TestValueMap_Yaml(t *testing.T) {
	vim = mdl.ValueInterfaceMap{}
	data := readFile(t, "testdata/raw.json")
	err := yaml.Unmarshal(data, &vim)

	if err != nil {
		assert.Fail(t, "could not unmarshal yaml")
	}

	str, ok = vim.GetString("a")
	assert.True(t, ok)
	assert.Equal(t, "b", str)

	vim, ok = vim.GetMap("map")
	assert.True(t, ok)
	assert.IsType(t, mdl.ValueInterfaceMap{}, vim)
}
