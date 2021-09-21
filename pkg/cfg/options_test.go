package cfg_test

import (
	"testing"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/mapx"
	"github.com/stretchr/testify/suite"
)

type OptionsTestSuite struct {
	suite.Suite
	config cfg.GosoConf
}

func (s *OptionsTestSuite) apply(options ...cfg.Option) {
	if err := s.config.Option(options...); err != nil {
		s.FailNowf(err.Error(), "can not apply options")
	}
}

func (s *OptionsTestSuite) SetupTest() {
	s.config = cfg.New()
}

func (s *OptionsTestSuite) TestWithConfigMap() {
	s.apply(cfg.WithConfigMap(map[string]interface{}{
		"b": true,
	}))

	actual := s.config.Get("b")
	s.Equal(true, actual)
}

func (s *OptionsTestSuite) TestWithConfigSetting() {
	expected := mapx.NewMapX(map[string]interface{}{
		"b": map[string]interface{}{
			"c1": map[string]interface{}{
				"i": 1,
				"s": "string",
			},
			"sl": []interface{}{
				map[string]interface{}{
					"b": true,
				},
				map[string]interface{}{
					"b": false,
				},
			},
		},
	})

	s.apply(cfg.WithConfigSetting("a.b.c1", map[string]interface{}{
		"i": 1,
		"s": "string",
	}))
	s.apply(cfg.WithConfigSetting("a.b.sl[0]", map[string]interface{}{
		"b": true,
	}))
	s.apply(cfg.WithConfigSetting("a.b.sl[1]", map[string]interface{}{
		"b": false,
	}))

	actual := s.config.Get("a")
	expectedMsi := expected.Msi()
	s.Equal(expectedMsi, actual)

	expected.Set("b.c2", map[string]interface{}{
		"b": true,
	})

	s.apply(cfg.WithConfigSetting("a.b.c2", map[string]interface{}{
		"b": true,
	}))

	actual = s.config.Get("a")
	expectedMsi = expected.Msi()
	s.Equal(expectedMsi, actual)
}

func TestOptionsTestSuite(t *testing.T) {
	suite.Run(t, new(OptionsTestSuite))
}
