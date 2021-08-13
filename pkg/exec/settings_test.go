package exec_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/exec"
	"github.com/stretchr/testify/suite"
)

type SettingsTestSuite struct {
	suite.Suite
	config cfg.GosoConf
}

func (s *SettingsTestSuite) SetupSuite() {
	s.config = cfg.New()
}

func (s *SettingsTestSuite) setupConfig(file string) {
	path := fmt.Sprintf("testdata/settings_%s.yml", file)

	if err := s.config.Option(cfg.WithConfigFile(path, "yaml")); err != nil {
		s.FailNow(err.Error(), "can not initialize config")
	}
}

func (s *SettingsTestSuite) TestDefault() {
	settings := exec.ReadBackoffSettings(s.config)
	expected := exec.BackoffSettings{
		InitialInterval: time.Millisecond * 50,
		MaxAttempts:     10,
		MaxElapsedTime:  time.Minute * 10,
		MaxInterval:     time.Second * 10,
	}

	s.Equal(expected, settings)
}

func (s *SettingsTestSuite) TestOnce() {
	s.setupConfig("once")

	settings := exec.ReadBackoffSettings(s.config)
	expected := exec.BackoffSettings{
		MaxAttempts: 1,
	}

	s.Equal(expected, settings)
}

func (s *SettingsTestSuite) TestInfinite() {
	s.setupConfig("infinite")

	settings := exec.ReadBackoffSettings(s.config)
	expected := exec.BackoffSettings{
		InitialInterval: time.Millisecond * 50,
		MaxAttempts:     0,
		MaxElapsedTime:  0,
		MaxInterval:     time.Second * 10,
	}

	s.Equal(expected, settings)
}

func (s *SettingsTestSuite) TestMultiplePathTypes() {
	s.setupConfig("multiple_path_types")

	settings := exec.ReadBackoffSettings(s.config, "ddb")
	expected := exec.BackoffSettings{
		MaxAttempts:    1,
		MaxElapsedTime: 0,
	}
	s.Equal(expected, settings)

	settings = exec.ReadBackoffSettings(s.config, "cloud.aws")
	expected = exec.BackoffSettings{
		InitialInterval: time.Millisecond * 50,
		MaxAttempts:     0,
		MaxElapsedTime:  0,
		MaxInterval:     time.Second * 10,
	}
	s.Equal(expected, settings)

	settings = exec.ReadBackoffSettings(s.config, "cloud2.aws", "cloud.aws", "ddb")
	expected = exec.BackoffSettings{
		CancelDelay:     time.Second,
		InitialInterval: time.Second * 2,
		MaxAttempts:     3,
		MaxElapsedTime:  time.Minute * 4,
		MaxInterval:     time.Second * 5,
	}
	s.Equal(expected, settings)
}

func (s *SettingsTestSuite) TestMissingStep() {
	s.setupConfig("missing_step")

	settings := exec.ReadBackoffSettings(s.config, "redis.default")
	expected := exec.BackoffSettings{
		InitialInterval: time.Millisecond * 50,
		MaxAttempts:     0,
		MaxElapsedTime:  0,
		MaxInterval:     time.Second * 10,
	}
	s.Equal(expected, settings)
}

func TestSettingsTestSuite(t *testing.T) {
	suite.Run(t, new(SettingsTestSuite))
}
