package redis_test

import (
	"testing"
	"time"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/exec"
	"github.com/justtrackio/gosoline/pkg/redis"
	"github.com/stretchr/testify/suite"
)

func TestFactoryTestSuite(t *testing.T) {
	suite.Run(t, new(SettingsTestSuite))
}

type SettingsTestSuite struct {
	suite.Suite
	config cfg.GosoConf
}

func (s *SettingsTestSuite) SetupTest() {
	s.config = cfg.New()
}

func (s *SettingsTestSuite) initConfig(settings map[string]any) {
	appIdConfig := cfg.WithConfigMap(map[string]any{
		"app": map[string]any{
			"env":  "env",
			"name": "redis",
			"tags": map[string]any{
				"project": "gosoline",
				"family":  "fam",
				"group":   "grp",
			},
		},
	})

	if err := s.config.Option(cfg.WithConfigMap(settings), appIdConfig); err != nil {
		s.FailNow(err.Error(), "can not setup config values")
	}
}

func (s *SettingsTestSuite) TestDefault() {
	s.initConfig(map[string]any{})

	settings, err := redis.ReadSettings(s.config, "default")
	s.NoError(err, "there should be no error reading the settings")

	expected := &redis.Settings{
		Identity: cfg.Identity{
			Name: "redis",
			Env:  "env",
			Tags: cfg.Tags{
				"project": "gosoline",
				"family":  "fam",
				"group":   "grp",
			},
		},
		Name: "default",
		Naming: redis.Naming{
			AddressPattern:   "{name}.{app.tags.group}.redis.{app.env}.{app.tags.family}",
			AddressDelimiter: ".",
			KeyPattern:       "{key}",
			KeyDelimiter:     "-",
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

func (s *SettingsTestSuite) TestDedicated() {
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
		Identity: cfg.Identity{
			Name: "redis",
			Env:  "env",
			Tags: cfg.Tags{
				"project": "gosoline",
				"family":  "fam",
				"group":   "grp",
			},
		},
		Name: "dedicated",
		Naming: redis.Naming{
			AddressPattern:   "{name}.{app.tags.group}.redis.{app.env}.{app.tags.family}",
			AddressDelimiter: ".",
			KeyPattern:       "{key}",
			KeyDelimiter:     "-",
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

func (s *SettingsTestSuite) TestWithDefaults() {
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
		Identity: cfg.Identity{
			Name: "redis",
			Env:  "env",
			Tags: cfg.Tags{
				"project": "gosoline",
				"family":  "fam",
				"group":   "grp",
			},
		},
		Name: "partial",
		Naming: redis.Naming{
			AddressPattern:   "{name}.{app.tags.group}.redis.{app.env}.{app.tags.family}",
			AddressDelimiter: ".",
			KeyPattern:       "{key}",
			KeyDelimiter:     "-",
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

func (s *SettingsTestSuite) TestKeyNamingPattern() {
	// Test that default key naming pattern is set correctly
	s.initConfig(map[string]any{})

	settings, err := redis.ReadSettings(s.config, "default")
	s.NoError(err, "there should be no error reading the settings")
	s.Equal("{key}", settings.Naming.KeyPattern)

	// Test custom key naming pattern
	s.initConfig(map[string]any{
		"redis": map[string]any{
			"default": map[string]any{
				"naming": map[string]any{
					"key_pattern": "{app.env}-{app.name}-{key}",
				},
			},
		},
	})

	settings, err = redis.ReadSettings(s.config, "default")
	s.NoError(err, "there should be no error reading the settings")
	s.Equal("{app.env}-{app.name}-{key}", settings.Naming.KeyPattern)
}
