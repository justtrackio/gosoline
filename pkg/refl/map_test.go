package refl_test

import (
	"github.com/applike/gosoline/pkg/refl"
	"github.com/stretchr/testify/assert"
	"testing"
)

type Item struct {
	Value string
}

func TestInterfaceToMapInterfaceInterface(t *testing.T) {
	items := map[int]Item{
		1: {
			Value: "foo",
		},
		2: {
			Value: "bar",
		},
	}

	mii, err := refl.InterfaceToMapInterfaceInterface(items)

	assert.NoError(t, err)
	assert.Len(t, mii, 2)

	assert.IsType(t, Item{}, mii[1])
	assert.IsType(t, Item{}, mii[2])

	assert.Equal(t, "foo", mii[1].(Item).Value)
	assert.Equal(t, "bar", mii[2].(Item).Value)
}

func TestMapOf(t *testing.T) {
	items := make(map[int]Item)
	m, err := refl.MapOf(items)

	assert.NoError(t, err)

	item := m.NewElement().(*Item)
	item.Value = "foo"
	m.Set(3, item)

	item = m.NewElement().(*Item)
	item.Value = "bar"
	m.Set(5, item)

	assert.Len(t, items, 2)
	assert.Equal(t, "foo", items[3].Value)
	assert.Equal(t, "bar", items[5].Value)
}
