package redis_test

import (
	"testing"
	"time"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/exec"
	"github.com/justtrackio/gosoline/pkg/redis"
	"github.com/stretchr/testify/suite"
)

type FactoryTestSuite struct {
	suite.Suite
	config cfg.GosoConf
}

func (s *FactoryTestSuite) SetupTest() {
	s.config = cfg.New()
}

func (s *FactoryTestSuite) initConfig(settings map[string]any) {
	appIdConfig := cfg.WithConfigMap(map[string]any{
		"app_project": "gosoline",
		"app_family":  "fam",
		"app_group":   "grp",
		"app_name":    "redis",
		"env":         "env",
	})

	if err := s.config.Option(cfg.WithConfigMap(settings), appIdConfig); err != nil {
		s.FailNow(err.Error(), "can not setup config values")
	}
}

func (s *FactoryTestSuite) TestDefault() {
	s.initConfig(map[string]any{})

	settings, err := redis.ReadSettings(s.config, "default")
	s.NoError(err, "there should be no error reading the settings")

	expected := &redis.Settings{
		AppId: cfg.AppId{
			Project:     "gosoline",
			Environment: "env",
			Family:      "fam",
			Group:       "grp",
			Application: "redis",
			Realm:       "gosoline-env-fam-grp",
		},
		Name: "default",
		Naming: redis.Naming{
			Pattern: "{name}.{group}.redis.{env}.{family}",
		},
		Dialer:  "tcp",
		Address: "127.0.0.1:6379",
		BackoffSettings: exec.BackoffSettings{
			InitialInterval: 50 * time.Millisecond,
			MaxAttempts:     10,
			MaxInterval:     time.Second * 10,
			MaxElapsedTime:  time.Minute * 10,
		},
	}

	s.Equal(expected, settings)
}

func (s *FactoryTestSuite) TestDedicated() {
	s.initConfig(map[string]any{
		"redis": map[string]any{
			"dedicated": map[string]any{
				"dialer":  "srv",
				"address": "dedicated.address",
				"backoff": map[string]any{
					"max_elapsed_time": "1m",
				},
			},
		},
	})

	settings, err := redis.ReadSettings(s.config, "dedicated")
	s.NoError(err, "there should be no error reading the settings")

	expected := &redis.Settings{
		AppId: cfg.AppId{
			Project:     "gosoline",
			Environment: "env",
			Family:      "fam",
			Group:       "grp",
			Application: "redis",
			Realm:       "gosoline-env-fam-grp",
		},
		Name: "dedicated",
		Naming: redis.Naming{
			Pattern: "{name}.{group}.redis.{env}.{family}",
		},
		Dialer:  "srv",
		Address: "dedicated.address",
		BackoffSettings: exec.BackoffSettings{
			InitialInterval: 50 * time.Millisecond,
			MaxAttempts:     10,
			MaxInterval:     time.Second * 10,
			MaxElapsedTime:  time.Minute,
		},
	}

	s.Equal(expected, settings)
}

func (s *FactoryTestSuite) TestWithDefaults() {
	s.initConfig(map[string]any{
		"redis": map[string]any{
			"default": map[string]any{
				"dialer": "srv",
				"backoff": map[string]any{
					"max_elapsed_time": "1m",
				},
			},
			"partial": map[string]any{
				"address": "partial.address",
			},
		},
	})

	settings, err := redis.ReadSettings(s.config, "partial")
	s.NoError(err, "there should be no error reading the settings")

	expected := &redis.Settings{
		AppId: cfg.AppId{
			Project:     "gosoline",
			Environment: "env",
			Family:      "fam",
			Group:       "grp",
			Application: "redis",
			Realm:       "gosoline-env-fam-grp",
		},
		Name: "partial",
		Naming: redis.Naming{
			Pattern: "{name}.{group}.redis.{env}.{family}",
		},
		Dialer:  "srv",
		Address: "partial.address",
		BackoffSettings: exec.BackoffSettings{
			InitialInterval: 50 * time.Millisecond,
			MaxAttempts:     10,
			MaxInterval:     time.Second * 10,
			MaxElapsedTime:  time.Minute,
		},
	}

	s.Equal(expected, settings)
}

func TestFactoryTestSuite(t *testing.T) {
	suite.Run(t, new(FactoryTestSuite))
}
