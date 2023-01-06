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

func (s *FactoryTestSuite) initConfig(settings map[string]interface{}) {
	appIdConfig := cfg.WithConfigMap(map[string]interface{}{
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
	s.initConfig(map[string]interface{}{})

	settings := redis.ReadSettings(s.config, "default")

	expected := &redis.Settings{
		AppId: cfg.AppId{
			Project:     "gosoline",
			Environment: "env",
			Family:      "fam",
			Group:       "grp",
			Application: "redis",
		},
		Name:    "default",
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
	s.initConfig(map[string]interface{}{
		"redis": map[string]interface{}{
			"dedicated": map[string]interface{}{
				"dialer":  "srv",
				"address": "dedicated.address",
				"backoff": map[string]interface{}{
					"max_elapsed_time": "1m",
				},
			},
		},
	})

	settings := redis.ReadSettings(s.config, "dedicated")

	expected := &redis.Settings{
		AppId: cfg.AppId{
			Project:     "gosoline",
			Environment: "env",
			Family:      "fam",
			Group:       "grp",
			Application: "redis",
		},
		Name:    "dedicated",
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
	s.initConfig(map[string]interface{}{
		"redis": map[string]interface{}{
			"default": map[string]interface{}{
				"dialer": "srv",
				"backoff": map[string]interface{}{
					"max_elapsed_time": "1m",
				},
			},
			"partial": map[string]interface{}{
				"address": "partial.address",
			},
		},
	})

	settings := redis.ReadSettings(s.config, "partial")

	expected := &redis.Settings{
		AppId: cfg.AppId{
			Project:     "gosoline",
			Environment: "env",
			Family:      "fam",
			Group:       "grp",
			Application: "redis",
		},
		Name:    "partial",
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
