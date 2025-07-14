package fixtures

import (
	"context"
	"fmt"

	"github.com/justtrackio/gosoline/pkg/appctx"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
)

type (
	containerContextKey struct{}
	Container           struct {
		fixtureSetFactories    map[string][]FixtureSetsFactory
		postProcessorFactories []PostProcessorFactory
	}
)

func ProvideContainer(ctx context.Context) (*Container, error) {
	return appctx.Provide(ctx, containerContextKey{}, func() (*Container, error) {
		return &Container{
			fixtureSetFactories:    map[string][]FixtureSetsFactory{},
			postProcessorFactories: []PostProcessorFactory{},
		}, nil
	})
}

func AddFixtureSetFactory(ctx context.Context, group string, factory FixtureSetsFactory) error {
	var err error
	var container *Container

	if container, err = ProvideContainer(ctx); err != nil {
		return fmt.Errorf("could not get fixtures container: %w", err)
	}

	container.AddFixtureSetFactory(group, factory)

	return nil
}

func AddFixtureSetPostProcessorFactory(ctx context.Context, factories ...PostProcessorFactory) error {
	var err error
	var container *Container

	if container, err = ProvideContainer(ctx); err != nil {
		return fmt.Errorf("could not get fixtures container: %w", err)
	}

	container.AddPostProcessorFactories(factories...)

	return nil
}

func (c *Container) AddFixtureSetFactory(group string, factory FixtureSetsFactory) {
	c.fixtureSetFactories[group] = append(c.fixtureSetFactories[group], factory)
}

func (c *Container) AddPostProcessorFactories(factory ...PostProcessorFactory) {
	c.postProcessorFactories = append(c.postProcessorFactories, factory...)
}

func (c *Container) Build(ctx context.Context, config cfg.Config, logger log.Logger) (FixtureLoader, error) {
	enabled, err := isFixtureLoadingEnabled(config)
	if err != nil {
		return nil, fmt.Errorf("could not check if fixture loading is enabled: %w", err)
	}

	if !enabled {
		return NewFixtureLoaderDisabled(logger), nil
	}

	var fixtureSets []FixtureSet
	var loader FixtureLoader
	postProcessors := make([]PostProcessor, len(c.postProcessorFactories))
	allFixtureSets := make(map[string][]FixtureSet)

	for group, factories := range c.fixtureSetFactories {
		for _, factory := range factories {
			if fixtureSets, err = factory(ctx, config, logger, group); err != nil {
				return nil, fmt.Errorf("could not build fixture sets for group %q: %w", group, err)
			}

			allFixtureSets[group] = append(allFixtureSets[group], fixtureSets...)
		}
	}

	for i, factory := range c.postProcessorFactories {
		if postProcessors[i], err = factory(ctx, config, logger); err != nil {
			return nil, fmt.Errorf("can not build fixture post processor: %w", err)
		}
	}

	if loader, err = NewFixtureLoader(ctx, config, logger, allFixtureSets, postProcessors); err != nil {
		return nil, fmt.Errorf("could not build fixture loader: %w", err)
	}

	c.fixtureSetFactories = map[string][]FixtureSetsFactory{}
	c.postProcessorFactories = []PostProcessorFactory{}

	return loader, nil
}
