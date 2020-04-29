package cfg_test

import (
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/stretchr/testify/suite"
	"testing"
)

type MapTestSuite struct {
	suite.Suite
	m cfg.Map
}

func (s *MapTestSuite) SetupTest() {
	s.m = cfg.NewMap()
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

	expected := map[string]interface{}{
		"i": 1,
		"a": map[string]interface{}{
			"b": map[string]interface{}{
				"sl1": []int{1, 2},
			},
		},
		"sl2": []int{3, 4, 0, 7, 0, 9},
		"sl3": []interface{}{
			map[string]interface{}{},
			map[string]interface{}{
				"b": true,
			},
		},
		"m1": map[string]interface{}{
			"b": true,
		},
	}

	actual := s.m.Msi()
	s.Equal(expected, actual)
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

	msi := cfg.Map(data)

	s.Equal(1, msi.Get("i"))
	s.Equal("string", msi.Get("a.b.s"))

	s.Equal(map[string]interface{}{
		"b": true,
		"s": "string",
	}, msi.Get("msi"))

	s.Equal([]interface{}{1, 2}, msi.Get("sl1"))
	s.Equal(2, msi.Get("sl1[1]"))
	s.Equal(nil, msi.Get("sl1[2]"))
}

func TestMapTestSuite(t *testing.T) {
	suite.Run(t, new(MapTestSuite))
}
