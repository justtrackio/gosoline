package redis_test

import (
	"testing"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/redis"
	"github.com/stretchr/testify/suite"
)

type NamingTestSuite struct {
	suite.Suite
	config   cfg.GosoConf
	identity cfg.AppIdentity
}

func (s *NamingTestSuite) SetupTest() {
	s.config = cfg.New()
	s.identity = cfg.AppIdentity{
		Name: "myapp",
		Env:  "test",
		Tags: cfg.AppTags{
			"project": "myproject",
			"family":  "myfamily",
			"group":   "mygroup",
		},
	}
}

func (s *NamingTestSuite) initConfig(settings map[string]any) {
	appIdConfig := cfg.WithConfigMap(map[string]any{
		"app": map[string]any{
			"env":  "test",
			"name": "myapp",
			"tags": map[string]any{
				"project": "myproject",
				"family":  "myfamily",
				"group":   "mygroup",
			},
		},
	})

	if err := s.config.Option(cfg.WithConfigMap(settings), appIdConfig); err != nil {
		s.FailNow(err.Error(), "can not setup config values")
	}
}

func (s *NamingTestSuite) TestBuildFullyQualifiedKey_DefaultPattern() {
	s.initConfig(map[string]any{})

	key, err := redis.BuildFullyQualifiedKey(s.config, s.identity, "mykey")
	s.NoError(err)
	s.Equal("myproject-test-myfamily-mygroup-myapp-mykey", key)
}

func (s *NamingTestSuite) TestBuildFullyQualifiedKey_CustomMinimalPattern() {
	s.initConfig(map[string]any{
		"redis": map[string]any{
			"default": map[string]any{
				"naming": map[string]any{
					"key_pattern": "{app.env}-{app.name}-{key}",
				},
			},
		},
	})

	key, err := redis.BuildFullyQualifiedKey(s.config, s.identity, "mykey")
	s.NoError(err)
	s.Equal("test-myapp-mykey", key)
}

func (s *NamingTestSuite) TestBuildFullyQualifiedKey_CustomPatternWithSomeTags() {
	s.initConfig(map[string]any{
		"redis": map[string]any{
			"default": map[string]any{
				"naming": map[string]any{
					"key_pattern": "{app.tags.project}-{app.env}-{key}",
				},
			},
		},
	})

	key, err := redis.BuildFullyQualifiedKey(s.config, s.identity, "mykey")
	s.NoError(err)
	s.Equal("myproject-test-mykey", key)
}

func (s *NamingTestSuite) TestBuildFullyQualifiedKey_MissingKeyPlaceholder() {
	s.initConfig(map[string]any{
		"redis": map[string]any{
			"default": map[string]any{
				"naming": map[string]any{
					"key_pattern": "{app.env}-{app.name}", // Missing {key}
				},
			},
		},
	})

	_, err := redis.BuildFullyQualifiedKey(s.config, s.identity, "mykey")
	s.Error(err)
	s.Contains(err.Error(), "must contain {key} placeholder")
}

func (s *NamingTestSuite) TestBuildFullyQualifiedKey_MissingRequiredTag() {
	s.initConfig(map[string]any{
		"redis": map[string]any{
			"default": map[string]any{
				"naming": map[string]any{
					"key_pattern": "{app.tags.project}-{app.tags.missing}-{key}",
				},
			},
		},
	})

	_, err := redis.BuildFullyQualifiedKey(s.config, s.identity, "mykey")
	s.Error(err)
	s.Contains(err.Error(), "there is no config setting or default for key \"app.tags.missing\"")
}

func (s *NamingTestSuite) TestBuildFullyQualifiedKey_InvalidPlaceholder() {
	s.initConfig(map[string]any{
		"redis": map[string]any{
			"default": map[string]any{
				"naming": map[string]any{
					"key_pattern": "{app.invalid}-{key}",
				},
			},
		},
	})

	_, err := redis.BuildFullyQualifiedKey(s.config, s.identity, "mykey")
	s.Error(err)
	s.Contains(err.Error(), "there is no config setting or default for key \"app.invalid\"")
}

func (s *NamingTestSuite) TestBuildFullyQualifiedKey_ComplexKey() {
	s.initConfig(map[string]any{})

	key, err := redis.BuildFullyQualifiedKey(s.config, s.identity, "cache:user:123")
	s.NoError(err)
	s.Equal("myproject-test-myfamily-mygroup-myapp-cache:user:123", key)
}

func TestNamingTestSuite(t *testing.T) {
	suite.Run(t, new(NamingTestSuite))
}
