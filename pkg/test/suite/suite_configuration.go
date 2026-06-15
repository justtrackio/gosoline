package suite

import (
	"context"
	"maps"
	"slices"

	"github.com/justtrackio/gosoline/pkg/application"
	"github.com/justtrackio/gosoline/pkg/fixtures"
	"github.com/justtrackio/gosoline/pkg/kernel"
	"github.com/justtrackio/gosoline/pkg/test/env"
)

type SuiteConfiguration struct {
	envOptions  []env.Option
	envSetup    []func() error
	envIsShared bool

	fixtureSetFactories              []fixtures.FixtureSetsFactory
	fixtureSetPostProcessorFactories []fixtures.PostProcessorFactory

	appCtx       context.Context
	appOptions   []application.Option
	appModules   map[string]kernel.ModuleFactory
	appFactories []kernel.ModuleMultiFactory

	testCaseWhitelist   []string
	testCaseRepeatCount int
}

func (s *SuiteConfiguration) clone() *SuiteConfiguration {
	clone := &SuiteConfiguration{
		envOptions:                       append([]env.Option(nil), s.envOptions...),
		envSetup:                         append([]func() error(nil), s.envSetup...),
		envIsShared:                      s.envIsShared,
		fixtureSetFactories:              append([]fixtures.FixtureSetsFactory(nil), s.fixtureSetFactories...),
		fixtureSetPostProcessorFactories: append([]fixtures.PostProcessorFactory(nil), s.fixtureSetPostProcessorFactories...),
		appCtx:                           s.appCtx,
		appOptions:                       append([]application.Option(nil), s.appOptions...),
		appModules:                       maps.Clone(s.appModules),
		appFactories:                     append([]kernel.ModuleMultiFactory(nil), s.appFactories...),
		testCaseWhitelist:                append([]string(nil), s.testCaseWhitelist...),
		testCaseRepeatCount:              s.testCaseRepeatCount,
	}

	return clone
}

func newSuiteConfiguration(options []Option) *SuiteConfiguration {
	conf := &SuiteConfiguration{
		envOptions:          make([]env.Option, 0),
		envSetup:            make([]func() error, 0),
		appOptions:          make([]application.Option, 0),
		appModules:          make(map[string]kernel.ModuleFactory),
		appFactories:        make([]kernel.ModuleMultiFactory, 0),
		testCaseRepeatCount: 1,
	}

	for _, opt := range options {
		opt(conf)
	}

	return conf
}

func (s *SuiteConfiguration) addAppOption(opt application.Option) {
	s.appOptions = append(s.appOptions, opt)
}

func (s *SuiteConfiguration) addEnvOption(opt env.Option) {
	s.envOptions = append(s.envOptions, opt)
}

func (s *SuiteConfiguration) shouldSkip(name string) bool {
	return len(s.testCaseWhitelist) > 0 && !slices.Contains(s.testCaseWhitelist, name)
}
