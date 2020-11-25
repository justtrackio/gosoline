package test

import (
	"fmt"
	"github.com/applike/gosoline/pkg/application"
	"github.com/applike/gosoline/pkg/clock"
	"github.com/applike/gosoline/pkg/fixtures"
	"github.com/applike/gosoline/pkg/ipread"
	"github.com/applike/gosoline/pkg/kernel"
	"github.com/applike/gosoline/pkg/stream"
	"github.com/applike/gosoline/pkg/test/env"
	"github.com/spf13/cast"
	"os"
	"time"
)

type suiteOptions struct {
	envOptions   []env.Option
	envSetup     []func() error
	appOptions   []application.Option
	appModules   map[string]kernel.ModuleFactory
	appFactories []kernel.MultiModuleFactory
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

func WithClockProvider(clk clock.Clock) SuiteOption {
	return func(s *suiteOptions) {
		s.envSetup = append(s.envSetup, func() error {
			clock.Provider = clk
			return nil
		})
	}
}

func WithClockProviderAt(datetime string) SuiteOption {
	return func(s *suiteOptions) {
		s.envSetup = append(s.envSetup, func() error {
			var err error
			var dt time.Time

			if dt, err = cast.ToTimeE(datetime); err != nil {
				return fmt.Errorf("can not cast provided fake clock datetime %s: %w", datetime, err)
			}

			clock.Provider = clock.NewFakeClockAt(dt)

			return nil
		})
	}
}

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

func WithConsumer(callback stream.ConsumerCallbackFactory) SuiteOption {
	return WithModule("consumer-default", stream.NewConsumer("default", callback))
}

func WithContainerExpireAfter(expireAfter time.Duration) SuiteOption {
	return func(s *suiteOptions) {
		s.addEnvOption(env.WithContainerExpireAfter(expireAfter))
	}
}

func WithEnvSetup(setups ...func() error) SuiteOption {
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

func WithIpReadFromMemory(name string, records map[string]ipread.MemoryRecord) SuiteOption {
	provider := ipread.ProvideMemoryProvider(name)

	for ip, record := range records {
		provider.AddRecord(ip, record.CountryIso, record.CityName)
	}

	return func(s *suiteOptions) {
		key := fmt.Sprintf("ipread.%s.provider", name)
		s.addEnvOption(env.WithConfigSetting(key, "memory"))
	}
}

func WithModule(name string, module kernel.ModuleFactory) SuiteOption {
	return func(s *suiteOptions) {
		if s.appModules == nil {
			s.appModules = make(map[string]kernel.ModuleFactory)
		}

		s.appModules[name] = module
	}
}

func WithModuleFactory(factory kernel.MultiModuleFactory) SuiteOption {
	return func(s *suiteOptions) {
		s.appFactories = append(s.appFactories, factory)
	}
}

func WithoutAutoDetectedComponents(components ...string) SuiteOption {
	return func(s *suiteOptions) {
		s.addEnvOption(env.WithoutAutoDetectedComponents(components...))
	}
}
