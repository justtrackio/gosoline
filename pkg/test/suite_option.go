package test

import (
	"github.com/applike/gosoline/pkg/application"
	"github.com/applike/gosoline/pkg/fixtures"
	"github.com/applike/gosoline/pkg/kernel"
	"github.com/applike/gosoline/pkg/stream"
	"github.com/applike/gosoline/pkg/test/env"
	"os"
	"time"
)

type suiteOptions struct {
	envOptions   []env.Option
	envSetup     []func()
	appOptions   []application.Option
	appModules   map[string]kernel.Module
	appFactories []kernel.ModuleFactory
}

func (s *suiteOptions) addEnvOption(opt env.Option) {
	s.envOptions = append(s.envOptions, opt)
}

func (s *suiteOptions) addAppOption(opt application.Option) {
	s.appOptions = append(s.appOptions, opt)
}

func (s *suiteOptions) addOptionalTestConfig() {
	if _, err := os.Stat("config.test.yml"); os.IsNotExist(err) {
		return
	}

	s.addEnvOption(env.WithConfigFile("config.test.yml"))
}

type SuiteOption func(s *suiteOptions)

func WithConfigFile(file string) SuiteOption {
	return func(s *suiteOptions) {
		s.addEnvOption(env.WithConfigFile(file))
	}
}

func WithConfigMap(settings map[string]interface{}) SuiteOption {
	return func(s *suiteOptions) {
		s.addEnvOption(env.WithConfigMap(settings))
	}
}

func WithConsumer(callback stream.ConsumerCallback) SuiteOption {
	return WithModule("consumer-default", stream.NewConsumer("default", callback))
}

func WithContainerExpireAfter(expireAfter time.Duration) SuiteOption {
	return func(s *suiteOptions) {
		s.addEnvOption(env.WithContainerExpireAfter(expireAfter))
	}
}

func WithEnvSetup(setups ...func()) SuiteOption {
	return func(s *suiteOptions) {
		s.envSetup = append(s.envSetup, setups...)
	}
}

func WithFixtures(fixtureSets []*fixtures.FixtureSet) SuiteOption {
	return func(s *suiteOptions) {
		s.appOptions = append(s.appOptions, application.WithConfigSetting("fixtures", map[string]interface{}{
			"enabled": true,
		}))
		s.appOptions = append(s.appOptions, application.WithFixtures(fixtureSets))
	}
}

func WithLogLevel(level string) SuiteOption {
	return func(s *suiteOptions) {
		s.addEnvOption(env.WithLoggerLevel(level))
	}
}

func WithModule(name string, module kernel.Module) SuiteOption {
	return func(s *suiteOptions) {
		if s.appModules == nil {
			s.appModules = make(map[string]kernel.Module)
		}

		s.appModules[name] = module
	}
}

func WithModuleFactory(factory kernel.ModuleFactory) SuiteOption {
	return func(s *suiteOptions) {
		s.appFactories = append(s.appFactories, factory)
	}
}

func WithoutAutoDetectedComponents(components ...string) SuiteOption {
	return func(s *suiteOptions) {
		s.addEnvOption(env.WithoutAutoDetectedComponents(components...))
	}
}
