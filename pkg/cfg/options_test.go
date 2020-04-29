package cfg_test

import (
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/stretchr/testify/suite"
	"testing"
)

type OptionsTestSuite struct {
	suite.Suite
	config cfg.GosoConf
}

func (s *OptionsTestSuite) SetupTest() {
	s.config = cfg.New()
}

func (s *OptionsTestSuite) TestWithConfigSetting() {
	expected := cfg.Map(map[string]interface{}{
		"b": map[string]interface{}{
			"c1": map[string]interface{}{
				"i": 1,
				"s": "string",
			},
		},
	})
	err := s.config.Option(cfg.WithConfigSetting("a.b.c1", map[string]interface{}{
		"i": 1,
		"s": "string",
	}))
	s.NoError(err, "there should be no error")
	actual := s.config.Get("a")
	s.Equal(expected.Msi(), actual)

	expected.Set("b.c2", map[string]interface{}{
		"b": true,
	})
	err = s.config.Option(cfg.WithConfigSetting("a.b.c2", map[string]interface{}{
		"b": true,
	}))
	s.NoError(err, "there should be no error")
	actual = s.config.Get("a")
	s.Equal(expected.Msi(), actual)
}

func TestOptionsTestSuite(t *testing.T) {
	suite.Run(t, new(OptionsTestSuite))
}
