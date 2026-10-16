package cfg_test

import (
	"testing"

	"github.com/stretchr/testify/require"

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
	s.apply(cfg.WithConfigMap(map[string]any{
		"b": true,
	}))

	actual, err := s.config.Get("b")
	s.NoError(err)
	s.Equal(true, actual)
}

func (s *OptionsTestSuite) TestWithConfigSetting() {
	expected := mapx.NewMapX(map[string]any{
		"b": map[string]any{
			"c1": map[string]any{
				"i": 1,
				"s": "string",
			},
			"sl": []any{
				map[string]any{
					"b": true,
				},
				map[string]any{
					"b": false,
				},
			},
		},
	})

	s.apply(cfg.WithConfigSetting("a.b.c1", map[string]any{
		"i": 1,
		"s": "string",
	}))
	s.apply(cfg.WithConfigSetting("a.b.sl[0]", map[string]any{
		"b": true,
	}))
	s.apply(cfg.WithConfigSetting("a.b.sl[1]", map[string]any{
		"b": false,
	}))

	actual, err := s.config.Get("a")
	s.NoError(err)
	expectedMsi := expected.Msi()
	s.Equal(expectedMsi, actual)

	expected.Set("b.c2", map[string]any{
		"b": true,
	})

	s.apply(cfg.WithConfigSetting("a.b.c2", map[string]any{
		"b": true,
	}))

	actual, err = s.config.Get("a")
	s.NoError(err)
	expectedMsi = expected.Msi()
	s.Equal(expectedMsi, actual)
}

func TestOptionsTestSuite(t *testing.T) {
	suite.Run(t, new(OptionsTestSuite))
}

func TestWithConfigOverride(t *testing.T) {
	environment := cfg.NewMemoryEnvProvider(map[string]string{"STORE_BUCKET": "environment", "STORE_REGION": "eu-west-1"})
	config := cfg.NewWithInterfaces(environment)
	require.NoError(t, config.Option(cfg.WithEnvKeyReplacer(cfg.DefaultEnvKeyReplacer),
		cfg.WithConfigMap(map[string]any{"store.bucket": "file"}),
		cfg.WithConfigOverride("store.bucket", "isolated")))
	value, err := config.GetString("store.bucket")
	require.NoError(t, err)
	require.Equal(t, "isolated", value)
	settings := struct {
		Bucket string `cfg:"bucket"`
		Region string `cfg:"region"`
	}{}
	require.NoError(t, config.UnmarshalKey("store", &settings))
	require.Equal(t, "isolated", settings.Bucket)
	require.Equal(t, "eu-west-1", settings.Region)
	other := cfg.NewWithInterfaces(environment)
	require.NoError(t, other.Option(cfg.WithEnvKeyReplacer(cfg.DefaultEnvKeyReplacer)))
	value, err = other.GetString("store.bucket")
	require.NoError(t, err)
	require.Equal(t, "environment", value)
}
