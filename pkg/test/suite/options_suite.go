package suite

import (
	"fmt"
	"slices"
	"time"

	"github.com/justtrackio/gosoline/pkg/application"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/clock"
	"github.com/justtrackio/gosoline/pkg/fixtures"
	"github.com/justtrackio/gosoline/pkg/ipread"
	"github.com/justtrackio/gosoline/pkg/kernel"
	"github.com/justtrackio/gosoline/pkg/mdlsub"
	"github.com/justtrackio/gosoline/pkg/stream"
	"github.com/justtrackio/gosoline/pkg/test/env"
	"github.com/spf13/cast"
)

type suiteOptions struct {
	envOptions  []env.Option
	envSetup    []func() error
	envIsShared bool

	fixtureSetFactories              []fixtures.FixtureSetsFactory
	fixtureSetPostProcessorFactories []fixtures.PostProcessorFactory

	appOptions   []application.Option
	appModules   map[string]kernel.ModuleFactory
	appFactories []kernel.ModuleMultiFactory

	testCaseWhitelist   []string
	testCaseRepeatCount int
}

func newSuiteOptions() *suiteOptions {
	return &suiteOptions{
		envOptions:          make([]env.Option, 0),
		envSetup:            make([]func() error, 0),
		appOptions:          make([]application.Option, 0),
		appModules:          make(map[string]kernel.ModuleFactory),
		appFactories:        make([]kernel.ModuleMultiFactory, 0),
		testCaseRepeatCount: 1,
	}
}

func (s *suiteOptions) addAppOption(opt application.Option) {
	s.appOptions = append(s.appOptions, opt)
}

func (s *suiteOptions) addEnvOption(opt env.Option) {
	s.envOptions = append(s.envOptions, opt)
}

func (s *suiteOptions) shouldSkip(name string) bool {
	return len(s.testCaseWhitelist) > 0 && !slices.Contains(s.testCaseWhitelist, name)
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

func WithConfigDebug(s *suiteOptions) {
	s.addAppOption(application.WithConfigDebug)
}

func WithConfigFile(file string) Option {
	return func(s *suiteOptions) {
		s.addEnvOption(env.WithConfigFile(file))
	}
}

func WithConfigMap(settings map[string]any) Option {
	return func(s *suiteOptions) {
		s.addEnvOption(env.WithConfigMap(settings))
	}
}

func WithContainerExpireAfter(expireAfter time.Duration) Option {
	return func(s *suiteOptions) {
		s.addEnvOption(env.WithContainerExpireAfter(expireAfter))
	}
}

func WithUntypedConsumer(callback stream.UntypedConsumerCallbackFactory) Option {
	return WithModule("consumer-default", stream.NewUntypedConsumer("default", callback))
}

func WithConsumer[M any](callback stream.ConsumerCallbackFactory[M]) Option {
	return WithModule("consumer-default", stream.NewConsumer("default", callback))
}

func WithEnvSetup(setups ...func() error) Option {
	return func(s *suiteOptions) {
		s.envSetup = append(s.envSetup, setups...)
	}
}

func WithFixtureSetFactory(factory fixtures.FixtureSetsFactory, postProcessorFactories ...fixtures.PostProcessorFactory) Option {
	return func(s *suiteOptions) {
		s.fixtureSetFactories = append(s.fixtureSetFactories, factory)
		s.fixtureSetPostProcessorFactories = append(s.fixtureSetPostProcessorFactories, postProcessorFactories...)
	}
}

func WithFixtureSetFactories(factories []fixtures.FixtureSetsFactory, postProcessorFactories ...fixtures.PostProcessorFactory) Option {
	return func(s *suiteOptions) {
		s.fixtureSetFactories = append(s.fixtureSetFactories, factories...)
		s.fixtureSetPostProcessorFactories = append(s.fixtureSetPostProcessorFactories, postProcessorFactories...)
	}
}

func WithLogLevel(level string) Option {
	return func(s *suiteOptions) {
		s.addEnvOption(env.WithLoggerLevel(level))
	}
}

func WithLogRecording() Option {
	return func(s *suiteOptions) {
		s.addEnvOption(env.WithLogRecording())
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

func WithModule(name string, essentialModule kernel.ModuleFactory) Option {
	return func(s *suiteOptions) {
		s.appModules[name] = essentialModule
	}
}

func WithModuleFactory(factory kernel.ModuleMultiFactory) Option {
	return func(s *suiteOptions) {
		s.appFactories = append(s.appFactories, factory)
	}
}

func WithSharedEnvironment() Option {
	return func(s *suiteOptions) {
		s.envIsShared = true
	}
}

func WithStreamConsumerRetryDisabled(s *suiteOptions) {
	s.addAppOption(application.WithConfigCallback(func(config cfg.GosoConf) error {
		consumerNames, err := stream.GetAllConsumerNames(config)
		if err != nil {
			return fmt.Errorf("can not get consumer names: %w", err)
		}

		for _, name := range consumerNames {
			key := fmt.Sprintf("%s.enabled", stream.ConfigurableConsumerRetryKey(name))
			if err := config.Option(cfg.WithConfigSetting(key, false)); err != nil {
				return fmt.Errorf("can not set option %s: %w", key, err)
			}
		}

		return nil
	}))
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

func WithDbRepoChangeHistory() Option {
	return func(s *suiteOptions) {
		s.addAppOption(application.WithDbRepoChangeHistory)
	}
}

func WithHttpServerShares() Option {
	return func(s *suiteOptions) {
		s.addAppOption(application.WithHttpServerShares)
	}
}

// WithTestCaseWhitelist returns an option which only runs the tests contained in the given whitelist. A test not in the
// whitelist is skipped instead, allowing you to easily run a single test (e.g., for debugging).
func WithTestCaseWhitelist(testCases ...string) Option {
	return func(s *suiteOptions) {
		s.testCaseWhitelist = testCases
	}
}

// WithTestCaseRepeatCount repeats the whole test suite the given number of times. This can be useful if a problem doesn't
// happen on every run (e.g., because it is timing dependent).
func WithTestCaseRepeatCount(repeatCount int) Option {
	return func(s *suiteOptions) {
		s.testCaseRepeatCount = repeatCount
	}
}
