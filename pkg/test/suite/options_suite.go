package suite

import (
	"fmt"
	"time"

	"github.com/applike/gosoline/pkg/application"
	"github.com/applike/gosoline/pkg/clock"
	"github.com/applike/gosoline/pkg/fixtures"
	"github.com/applike/gosoline/pkg/ipread"
	"github.com/applike/gosoline/pkg/kernel"
	"github.com/applike/gosoline/pkg/mdlsub"
	"github.com/applike/gosoline/pkg/stream"
	"github.com/applike/gosoline/pkg/test/env"
	"github.com/spf13/cast"
)

type suiteOptions struct {
	envOptions  []env.Option
	envSetup    []func() error
	envIsShared bool

	appOptions   []application.Option
	appModules   map[string]kernel.ModuleFactory
	appFactories []kernel.MultiModuleFactory
}

func newSuiteOptions() *suiteOptions {
	return &suiteOptions{
		envOptions:   make([]env.Option, 0),
		envSetup:     make([]func() error, 0),
		appOptions:   make([]application.Option, 0),
		appModules:   make(map[string]kernel.ModuleFactory),
		appFactories: make([]kernel.MultiModuleFactory, 0),
	}
}

func (s *suiteOptions) addAppOption(opt application.Option) {
	s.appOptions = append(s.appOptions, opt)
}

func (s *suiteOptions) addEnvOption(opt env.Option) {
	s.envOptions = append(s.envOptions, opt)
}

type Option func(s *suiteOptions)

func WithClockProvider(clk clock.Clock) Option {
	return func(s *suiteOptions) {
		s.envSetup = append(s.envSetup, func() error {
			clock.Provider = clk
			return nil
		})
	}
}

func WithClockProviderAt(datetime string) Option {
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

func WithComponent(settings env.ComponentBaseSettingsAware) Option {
	return func(s *suiteOptions) {
		s.addEnvOption(env.WithComponent(settings))
	}
}

func WithConfigFile(file string) Option {
	return func(s *suiteOptions) {
		s.addEnvOption(env.WithConfigFile(file))
	}
}

func WithConfigMap(settings map[string]interface{}) Option {
	return func(s *suiteOptions) {
		s.addEnvOption(env.WithConfigMap(settings))
	}
}

func WithContainerExpireAfter(expireAfter time.Duration) Option {
	return func(s *suiteOptions) {
		s.addEnvOption(env.WithContainerExpireAfter(expireAfter))
	}
}

func WithConsumer(callback stream.ConsumerCallbackFactory) Option {
	return WithModule("consumer-default", stream.NewConsumer("default", callback))
}

func WithEnvSetup(setups ...func() error) Option {
	return func(s *suiteOptions) {
		s.envSetup = append(s.envSetup, setups...)
	}
}

func WithFixtures(fixtureSets []*fixtures.FixtureSet) Option {
	return func(s *suiteOptions) {
		s.appOptions = append(s.appOptions, application.WithConfigSetting("fixtures", map[string]interface{}{
			"enabled": true,
		}))
		s.appOptions = append(s.appOptions, application.WithFixtures(fixtureSets))
	}
}

func WithLogLevel(level string) Option {
	return func(s *suiteOptions) {
		s.addEnvOption(env.WithLoggerLevel(level))
	}
}

func WithIpReadFromMemory(name string, records map[string]ipread.MemoryRecord) Option {
	provider := ipread.ProvideMemoryProvider(name)

	for ip, record := range records {
		provider.AddRecord(ip, record)
	}

	return func(s *suiteOptions) {
		key := fmt.Sprintf("ipread.%s.provider", name)
		s.addEnvOption(env.WithConfigSetting(key, "memory"))
	}
}

func WithModule(name string, module kernel.ModuleFactory) Option {
	return func(s *suiteOptions) {
		s.appModules[name] = module
	}
}

func WithModuleFactory(factory kernel.MultiModuleFactory) Option {
	return func(s *suiteOptions) {
		s.appFactories = append(s.appFactories, factory)
	}
}

func WithSharedEnvironment() Option {
	return func(s *suiteOptions) {
		s.envIsShared = true
	}
}

func WithSubscribers(transformerFactoryMap mdlsub.TransformerMapTypeVersionFactories) Option {
	subs := mdlsub.NewSubscriberFactory(transformerFactoryMap)

	return WithModuleFactory(subs)
}

func WithoutAutoDetectedComponents(components ...string) Option {
	return func(s *suiteOptions) {
		s.addEnvOption(env.WithoutAutoDetectedComponents(components...))
	}
}
