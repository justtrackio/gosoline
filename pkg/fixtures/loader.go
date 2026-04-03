//go:build fixtures

package fixtures

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/funk"
	"github.com/justtrackio/gosoline/pkg/log"
)

type fixtureLoader struct {
	logger         log.Logger
	fixtureSets    map[string][]FixtureSet
	postProcessors []PostProcessor
	settings       *fixtureLoaderSettings
	sharedState    *SharedState
}

type fixtureSetValidationItem struct {
	group      string
	fixtureSet FixtureSet
}

func NewFixtureLoader(ctx context.Context, config cfg.Config, logger log.Logger, fixtureSets map[string][]FixtureSet, postProcessors []PostProcessor) (FixtureLoader, error) {
	settings, err := unmarshalFixtureLoaderSettings(config)
	if err != nil {
		return nil, fmt.Errorf("could not unmarshal fixture loader settings: %w", err)
	}

	sharedState, err := ProvideSharedState(ctx)
	if err != nil {
		return nil, fmt.Errorf("could not provide shared fixture state: %w", err)
	}

	if err := validateFixtureSets(fixtureSets, settings.Groups); err != nil {
		return nil, err
	}

	return &fixtureLoader{
		logger:         logger.WithChannel("fixtures"),
		fixtureSets:    fixtureSets,
		postProcessors: postProcessors,
		settings:       settings,
		sharedState:    sharedState,
	}, nil
}

func validateFixtureSets(fixtureSets map[string][]FixtureSet, groups []string) error {
	activeFixtureSets, hasSharedFixtureSets := collectActiveFixtureSets(fixtureSets, groups)

	if !hasSharedFixtureSets {
		return nil
	}

	err := validateActiveFixtureSets(activeFixtureSets)

	if err == nil {
		return nil
	}

	return fmt.Errorf("invalid fixture resource metadata: %w", err)
}

func collectActiveFixtureSets(fixtureSets map[string][]FixtureSet, groups []string) ([]fixtureSetValidationItem, bool) {
	activeFixtureSets := make([]fixtureSetValidationItem, 0)
	hasSharedFixtureSets := false

	for group, sets := range fixtureSets {
		if !slices.Contains(groups, group) {
			continue
		}

		for _, fixtureSet := range sets {
			if !fixtureSetEnabled(fixtureSet) {
				continue
			}

			activeFixtureSets = append(activeFixtureSets, fixtureSetValidationItem{
				group:      group,
				fixtureSet: fixtureSet,
			})
			hasSharedFixtureSets = hasSharedFixtureSets || IsSharedFixtureSet(fixtureSet)
		}
	}

	return activeFixtureSets, hasSharedFixtureSets
}

func validateActiveFixtureSets(activeFixtureSets []fixtureSetValidationItem) error {
	errs := make([]error, 0)

	for _, item := range activeFixtureSets {
		if err := validateActiveFixtureSet(item); err != nil {
			errs = append(errs, err)
		}
	}

	return errors.Join(errs...)
}

func validateActiveFixtureSet(item fixtureSetValidationItem) error {
	fixtureSet := item.fixtureSet
	description := FixtureSetDescription(fixtureSet)

	if IsSharedFixtureSet(fixtureSet) {
		return validateSharedFixtureSet(item.group, fixtureSet, description)
	}

	if fixtureSetUsesUnmanagedWriter(fixtureSet) {
		return nil
	}

	if _, ok := FixtureSetResourceIds(fixtureSet); ok {
		return nil
	}

	return fmt.Errorf(
		"fixture group %s: fixture set %s is missing managed resources on its writer while shared fixtures are enabled; wrap the writer with fixtures.NewManagedFixtureWriter(...) or fixtures.NewUnmanagedFixtureWriter(...)",
		item.group,
		description,
	)
}

func validateSharedFixtureSet(group string, fixtureSet FixtureSet, description string) error {
	if fixtureSetUsesUnmanagedWriter(fixtureSet) {
		return fmt.Errorf(
			"fixture group %s: shared fixture set %s uses a writer without managed resources; wrap the writer with fixtures.NewManagedFixtureWriter(...)",
			group,
			description,
		)
	}

	if _, ok := FixtureSetResourceIds(fixtureSet); ok {
		return nil
	}

	return fmt.Errorf(
		"fixture group %s: shared fixture set %s is missing managed resources on its writer; wrap the writer with fixtures.NewManagedFixtureWriter(...)",
		group,
		description,
	)
}

func (f *fixtureLoader) Load(ctx context.Context) error {
	if !f.settings.Enabled {
		f.logger.Info(ctx, "fixture loader is not enabled")

		return nil
	}

	f.logger.Info(ctx, "loading fixtures")
	start := time.Now()
	defer func() {
		f.logger.Info(ctx, "done loading fixtures in %s", time.Since(start))
	}()

	for group, fixtureSets := range f.fixtureSets {
		if !slices.Contains(f.settings.Groups, group) {
			f.logger.Info(ctx, "fixture group %s is not enabled", group)

			continue
		}

		for _, fixtureSet := range fixtureSets {
			if f.shouldSkipSharedFixtureSet(fixtureSet) {
				f.logger.Info(ctx, "skipping shared fixture set %T as it was already loaded", fixtureSet)

				continue
			}

			f.logger.Info(ctx, "loading fixtures for set %T", fixtureSet)

			if err := fixtureSet.Write(ctx); err != nil {
				return fmt.Errorf("failed to write fixtures: %w", err)
			}

			f.markSharedFixtureSetLoaded(fixtureSet)
		}
	}

	for _, processor := range f.postProcessors {
		if err := processor.Process(ctx); err != nil {
			return fmt.Errorf("can not post process fixtures: %w", err)
		}
	}

	return nil
}

func (f *fixtureLoader) shouldSkipSharedFixtureSet(fixtureSet FixtureSet) bool {
	return IsSharedFixtureSet(fixtureSet) && f.sharedState.IsLoaded(SharedFixtureSetKey(fixtureSet))
}

func (f *fixtureLoader) markSharedFixtureSetLoaded(fixtureSet FixtureSet) {
	if IsSharedFixtureSet(fixtureSet) {
		f.sharedState.MarkLoaded(SharedFixtureSetKey(fixtureSet))
	}
}

func (f *fixtureLoader) MutableResourceIds() ([]string, bool) {
	resourceIds := funk.Set[string]{}

	for group, fixtureSets := range f.fixtureSets {
		if !slices.Contains(f.settings.Groups, group) {
			continue
		}

		for _, fixtureSet := range fixtureSets {
			if !fixtureSetEnabled(fixtureSet) {
				continue
			}

			if IsSharedFixtureSet(fixtureSet) {
				continue
			}

			if fixtureSetUsesUnmanagedWriter(fixtureSet) {
				continue
			}

			ids, ok := FixtureSetResourceIds(fixtureSet)
			if !ok {
				return nil, false
			}

			resourceIds.Add(ids...)
		}
	}

	return resourceIds.ToSlice(), true
}

func fixtureSetEnabled(fixtureSet FixtureSet) bool {
	enabledAware, ok := fixtureSet.(EnabledAware)
	if !ok {
		return true
	}

	return enabledAware.IsEnabled()
}

func fixtureSetUsesUnmanagedWriter(fixtureSet FixtureSet) bool {
	return FixtureSetHasNoManagedResources(fixtureSet)
}
