package mapx_test

import (
	"sync"
	"testing"

	"github.com/justtrackio/gosoline/pkg/mapx"
	"github.com/stretchr/testify/suite"
)

type MapTestSuite struct {
	suite.Suite
	m *mapx.MapX
}

func TestMapTestSuite(t *testing.T) {
	suite.Run(t, new(MapTestSuite))
}

func (s *MapTestSuite) SetupTest() {
	s.m = mapx.NewMapX()
}

func (s *MapTestSuite) TestHas() {
	s.m.Set("a", 1)
	actual := s.m.Has("a")
	s.True(actual)

	actual = s.m.Has("sl[1]")
	s.False(actual)
}

func (s *MapTestSuite) TestSet() {
	s.m.Set("i", 1)
	s.m.Set("a.b.sl1", []int{1, 2})
	s.m.Set("sl2[0]", 3)
	s.m.Set("sl2[1]", 4)
	s.m.Set("sl2[5]", 9)
	s.m.Set("sl2[3]", 7)
	s.m.Set("sl3[1].b", true)
	s.m.Set("m1", map[string]any{
		"b": true,
	})
	s.m.Set("m2.subM", nil)
	s.m.Set("m3.subM", nil)
	s.m.Set("m3.subM.i", 8)

	expected := map[string]any{
		"i": 1,
		"a": map[string]any{
			"b": map[string]any{
				"sl1": []any{1, 2},
			},
		},
		"sl2": []any{3, 4, 0, 7, 0, 9},
		"sl3": []any{
			map[string]any{},
			map[string]any{
				"b": true,
			},
		},
		"m1": map[string]any{
			"b": true,
		},
		"m2": map[string]any{
			"subM": nil,
		},
		"m3": map[string]any{
			"subM": map[string]any{
				"i": 8,
			},
		},
	}

	actual := s.m.Msi()
	s.Equal(expected, actual)
}

func (s *MapTestSuite) TestSetMap() {
	msi := map[string]any{
		"a": 1,
		"b": 2,
	}
	mapToSet := mapx.NewMapX(msi)

	s.m.Set("c", mapToSet)
	actual, err := s.m.Get("c").Msi()
	s.NoError(err)
	s.Equal(msi, actual)

	s.m.Set("d[0]", mapToSet)
	actual, err = s.m.Get("d[0]").Msi()
	s.NoError(err)
	s.Equal(msi, actual)

	s.m.Set("d[0]", mapToSet)
	actual, err = s.m.Get("d[0]").Msi()
	s.NoError(err)
	s.Equal(msi, actual)

	s.m.Set("d[2]", mapToSet)
	actual, err = s.m.Get("d[2]").Msi()
	s.NoError(err)
	s.Equal(msi, actual)
}

func (s *MapTestSuite) TestSetKeyWithDots() {
	s.m.Set(`key\.with\.dots`, "value")

	keys := s.m.Keys()
	s.Equal([]string{`key\.with\.dots`}, keys)

	actual := s.m.Get(`key\.with\.dots`).Data()
	s.Equal("value", actual)
}

func (s *MapTestSuite) TestSetSliceOffset() {
	s.m.Set("sl[1]", 1)

	expected := map[string]any{
		"sl": []any{nil, 1},
	}

	actual := s.m.Msi()
	s.Equal(expected, actual)
}

func (s *MapTestSuite) TestSetSliceOfMaps() {
	s.m.Set("sl", []any{
		map[string]any{
			"i": 1,
		},
	})

	isMap := s.m.Get("sl[0]").IsMap()
	s.True(isMap)

	s.m.Set("m", map[string]any{
		"sl": []any{
			map[string]any{
				"i": 1,
			},
		},
	})

	isMap = s.m.Get("m.sl[0]").IsMap()
	s.True(isMap)
}

func (s *MapTestSuite) TestSetSliceOfMapsDoesntModify() {
	mapsList := []any{
		map[string]any{
			"true":  true,
			"false": false,
		},
	}

	m := mapx.NewMapX()
	m.Set("maps", mapsList)

	s.Equal([]any{
		map[string]any{
			"true":  true,
			"false": false,
		},
	}, mapsList, "the argument shouldn't be modified")
	s.Equal(m.Msi(), map[string]any{
		"maps": []any{
			map[string]any{
				"true":  true,
				"false": false,
			},
		},
	})
}

func (s *MapTestSuite) TestSkipExisting() {
	s.m.Set("a", 1)
	s.m.Set("a", 2, mapx.SkipExisting)
	s.Equal(1, s.m.Get("a").Data())

	s.m.Set("sl[2]", 3)
	s.m.Set("sl[2]", 4, mapx.SkipExisting)
	s.Equal(3, s.m.Get("sl[2]").Data())
}

func (s *MapTestSuite) TestGet() {
	data := map[string]any{
		"i": 1,
		"a": map[string]any{
			"b": map[string]any{
				"s": "string",
			},
		},
		"msi": map[string]any{
			"b": true,
			"s": "string",
		},
		"sl1": []any{1, 2},
	}

	msi := mapx.NewMapX(data)

	s.Equal(1, msi.Get("i").Data())
	s.Equal(1, msi.Get(".i").Data())
	s.Equal("string", msi.Get("a.b.s").Data())

	act, err := msi.Get("msi").Msi()
	s.NoError(err)
	s.Equal(map[string]any{
		"b": true,
		"s": "string",
	}, act)

	s.Equal([]any{1, 2}, msi.Get("sl1").Data())
	s.Equal(2, msi.Get("sl1[1]").Data())
	s.Equal(nil, msi.Get("sl1[2]").Data())

	root := msi.Get(".").Data()
	s.Equal(data, root)
}

func (s *MapTestSuite) TestMergeRootEmpty() {
	s.m.Merge(".", map[string]any{})
	s.Empty(s.m.Msi())
}

func (s *MapTestSuite) TestMerge() {
	s.m.Set("b", true)
	s.m.Set("msi", map[string]any{
		"i":  1,
		"s1": "string1",
		"sl": []any{1, 2, 3},
	})
	s.m.Set("sl", []any{1, 2, 3})

	s.m.Merge(".", map[string]any{
		"msi": map[string]any{
			"f": 1.1,
		},
	})
	s.m.Merge("msi", map[string]any{
		"s2": "string2",
	})
	s.m.Merge("msi", map[string]any{
		"sl[3]": 4,
	})
	s.m.Merge("emptySl", []any{})

	expected := map[string]any{
		"b": true,
		"msi": map[string]any{
			"i":  1,
			"f":  1.1,
			"s1": "string1",
			"s2": "string2",
			"sl": []any{1, 2, 3, 4},
		},
		"sl":      []any{1, 2, 3},
		"emptySl": []any{},
	}

	msi := s.m.Msi()
	s.Equal(expected, msi)
}

func (s *MapTestSuite) TestMergeMap() {
	msi := map[string]any{
		"a":   1,
		"b":   2,
		"msi": map[string]any{},
	}
	mapToMerge := mapx.NewMapX(msi)

	s.m.Merge(".", mapToMerge)
	actual := s.m.Msi()

	s.Equal(msi, actual)
}

func (s *MapTestSuite) TestAppend() {
	err := s.m.Append("slice.at.a", "foo")
	s.NoError(err)

	current, err := s.m.Get("slice.at.a").StringSlice()
	s.NoError(err)
	s.Equal([]string{"foo"}, current)

	err = s.m.Append("slice.at.a", "bar", "baz")
	s.NoError(err)

	current, err = s.m.Get("slice.at.a").StringSlice()
	s.NoError(err)
	s.Equal([]string{"foo", "bar", "baz"}, current)
}

func (s *MapTestSuite) TestPreventExternalModify() {
	s.m.Set("myMap.foo", true)
	s.m.Set("myMap.bar", false)

	current, err := s.m.Get("myMap").Msi()
	s.NoError(err)
	s.Equal(map[string]any{
		"foo": true,
		"bar": false,
	}, current)

	myMap, err := s.m.Get("myMap").Map()
	s.NoError(err)
	myMap.Set("baz", "value")

	// no change expected
	current, err = s.m.Get("myMap").Msi()
	s.NoError(err)
	s.Equal(map[string]any{
		"foo": true,
		"bar": false,
	}, current)

	s.m.Set("myMap", myMap)

	// change expected
	current, err = s.m.Get("myMap").Msi()
	s.NoError(err)
	s.Equal(map[string]any{
		"foo": true,
		"bar": false,
		"baz": "value",
	}, current)
}

func (s *MapTestSuite) TestConcurrentModify() {
	s.m.Set("myMap.foo", 1)

	wg := sync.WaitGroup{}
	for i := 0; i < 1000; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			m, err := s.m.Get("myMap").Map()
			s.NoError(err)

			m.Set("foo", i)
		}()
	}

	wg.Wait()

	foo := s.m.Get("myMap.foo").Data().(int)
	s.True(foo >= 0 && foo < 1000)
}

func (s *MapTestSuite) TestConcurrentAppend() {
	s.m.Set("mySlice", []int{})

	wg := sync.WaitGroup{}
	for i := 0; i < 1000; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			err := s.m.Append("mySlice", i)
			s.NoError(err)
		}()
	}

	wg.Wait()

	mySlice, err := s.m.Get("mySlice").Slice()
	s.NoError(err)
	s.Len(mySlice, 1000)
}
