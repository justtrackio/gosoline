package mapx_test

import (
	"github.com/applike/gosoline/pkg/mapx"
	"github.com/stretchr/testify/suite"
	"testing"
)

type MapTestSuite struct {
	suite.Suite
	m *mapx.MapX
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
	s.m.Set("m1", map[string]interface{}{
		"b": true,
	})
	s.m.Set("m2.subM", nil)
	s.m.Set("m3.subM", nil)
	s.m.Set("m3.subM.i", 8)

	expected := map[string]interface{}{
		"i": 1,
		"a": map[string]interface{}{
			"b": map[string]interface{}{
				"sl1": []interface{}{1, 2},
			},
		},
		"sl2": []interface{}{3, 4, 0, 7, 0, 9},
		"sl3": []interface{}{
			map[string]interface{}{},
			map[string]interface{}{
				"b": true,
			},
		},
		"m1": map[string]interface{}{
			"b": true,
		},
		"m2": map[string]interface{}{
			"subM": nil,
		},
		"m3": map[string]interface{}{
			"subM": map[string]interface{}{
				"i": 8,
			},
		},
	}

	actual := s.m.Msi()
	s.Equal(expected, actual)
}

func (s *MapTestSuite) TestSetMap() {
	msi := map[string]interface{}{
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

func (s *MapTestSuite) TestSetSliceOffset() {
	s.m.Set("sl[1]", 1)

	expected := map[string]interface{}{
		"sl": []interface{}{nil, 1},
	}

	actual := s.m.Msi()
	s.Equal(expected, actual)
}

func (s *MapTestSuite) TestSetSliceOfMaps() {
	s.m.Set("sl", []interface{}{
		map[string]interface{}{
			"i": 1,
		},
	})

	isMap := s.m.Get("sl[0]").IsMap()
	s.True(isMap)

	s.m.Set("m", map[string]interface{}{
		"sl": []interface{}{
			map[string]interface{}{
				"i": 1,
			},
		},
	})

	isMap = s.m.Get("m.sl[0]").IsMap()
	s.True(isMap)
}

func (s *MapTestSuite) TestSetSliceOfMapsDoesntModify() {
	mapsList := []interface{}{
		map[string]interface{}{
			"true":  true,
			"false": false,
		},
	}

	m := mapx.NewMapX()
	m.Set("maps", mapsList)

	s.Equal([]interface{}{
		map[string]interface{}{
			"true":  true,
			"false": false,
		},
	}, mapsList, "the argument shouldn't be modified")
	s.Equal(m.Msi(), map[string]interface{}{
		"maps": []interface{}{
			map[string]interface{}{
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
	data := map[string]interface{}{
		"i": 1,
		"a": map[string]interface{}{
			"b": map[string]interface{}{
				"s": "string",
			},
		},
		"msi": map[string]interface{}{
			"b": true,
			"s": "string",
		},
		"sl1": []interface{}{1, 2},
	}

	msi := mapx.NewMapX(data)

	s.Equal(1, msi.Get("i").Data())
	s.Equal(1, msi.Get(".i").Data())
	s.Equal("string", msi.Get("a.b.s").Data())

	act, err := msi.Get("msi").Msi()
	s.NoError(err)
	s.Equal(map[string]interface{}{
		"b": true,
		"s": "string",
	}, act)

	s.Equal([]interface{}{1, 2}, msi.Get("sl1").Data())
	s.Equal(2, msi.Get("sl1[1]").Data())
	s.Equal(nil, msi.Get("sl1[2]").Data())
}

func (s *MapTestSuite) TestMergeRootEmpty() {
	s.m.Merge(".", map[string]interface{}{})
	s.Empty(s.m.Msi())
}

func (s *MapTestSuite) TestMerge() {
	s.m.Set("b", true)
	s.m.Set("msi", map[string]interface{}{
		"i":  1,
		"s1": "string1",
		"sl": []interface{}{1, 2, 3},
	})
	s.m.Set("sl", []interface{}{1, 2, 3})

	s.m.Merge(".", map[string]interface{}{
		"msi": map[string]interface{}{
			"f": 1.1,
		},
	})
	s.m.Merge("msi", map[string]interface{}{
		"s2": "string2",
	})
	s.m.Merge("msi", map[string]interface{}{
		"sl[3]": 4,
	})
	s.m.Merge("emptySl", []interface{}{})

	expected := map[string]interface{}{
		"b": true,
		"msi": map[string]interface{}{
			"i":  1,
			"f":  1.1,
			"s1": "string1",
			"s2": "string2",
			"sl": []interface{}{1, 2, 3, 4},
		},
		"sl":      []interface{}{1, 2, 3},
		"emptySl": []interface{}{},
	}

	msi := s.m.Msi()
	s.Equal(expected, msi)
}

func (s *MapTestSuite) TestMergeMap() {
	msi := map[string]interface{}{
		"a":   1,
		"b":   2,
		"msi": map[string]interface{}{},
	}
	mapToMerge := mapx.NewMapX(msi)

	s.m.Merge(".", mapToMerge)
	actual := s.m.Msi()

	s.Equal(msi, actual)
}

func TestMapTestSuite(t *testing.T) {
	suite.Run(t, new(MapTestSuite))
}
