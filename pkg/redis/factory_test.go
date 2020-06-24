package redis_test

import (
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/redis"
	"github.com/stretchr/testify/suite"
	"testing"
	"time"
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
		"app_family":  "test",
		"app_name":    "redis",
		"env":         "test",
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
			Environment: "test",
			Family:      "test",
			Application: "redis",
		},
		Name:    "default",
		Dialer:  "tcp",
		Address: "127.0.0.1:6379",
		BackoffSettings: redis.BackoffSettings{
			InitialInterval:     time.Second,
			RandomizationFactor: 0.2,
			Multiplier:          3.0,
			MaxInterval:         time.Second * 30,
			MaxElapsedTime:      0,
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
			Environment: "test",
			Family:      "test",
			Application: "redis",
		},
		Name:    "dedicated",
		Dialer:  "srv",
		Address: "dedicated.address",
		BackoffSettings: redis.BackoffSettings{
			InitialInterval:     time.Second,
			RandomizationFactor: 0.2,
			Multiplier:          3.0,
			MaxInterval:         time.Second * 30,
			MaxElapsedTime:      time.Minute,
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
			Environment: "test",
			Family:      "test",
			Application: "redis",
		},
		Name:    "partial",
		Dialer:  "srv",
		Address: "partial.address",
		BackoffSettings: redis.BackoffSettings{
			InitialInterval:     time.Second,
			RandomizationFactor: 0.2,
			Multiplier:          3.0,
			MaxInterval:         time.Second * 30,
			MaxElapsedTime:      time.Minute,
		},
	}

	s.Equal(expected, settings)
}

func TestFactoryTestSuite(t *testing.T) {
	suite.Run(t, new(FactoryTestSuite))
}
