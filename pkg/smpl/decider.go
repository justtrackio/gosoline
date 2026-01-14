package smpl

import (
	"context"
	"fmt"
	"slices"
	"strconv"

	"github.com/justtrackio/gosoline/pkg/appctx"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/metric"
	"github.com/justtrackio/gosoline/pkg/smpl/smplctx"
)

type (
	ctxKeyDecider struct{}

	//go:generate go run github.com/vektra/mockery/v2 --name Decider

	// Decider determines if a request should be sampled.
	Decider interface {
		// Decide evaluates the configured strategies to determine if the current context should be sampled.
		// It accepts additional strategies that take precedence over the configured ones.
		// It returns the context (potentially enriched with the sampling decision), the decision itself (true if sampled), and any error occurred.
		Decide(ctx context.Context, additionalStrategies ...Strategy) (context.Context, bool, error)
	}

	// Settings configures the sampling behavior.
	Settings struct {
		// Enabled toggles the sampling logic. If false, sampling is assumed to be true but not stored in the context.
		Enabled bool `cfg:"enabled" default:"false"`
		// Strategies is a list of strategy names to apply in order.
		Strategies []string `cfg:"strategies"`
	}
)

// ProvideDecider provides a singleton Decider instance from the application context.
func ProvideDecider(ctx context.Context, config cfg.Config) (Decider, error) {
	return appctx.Provide(ctx, ctxKeyDecider{}, func() (Decider, error) {
		return NewDecider(ctx, config)
	})
}

// NewDecider creates a new Decider based on the provided configuration.
// It parses the settings from the "sampling" configuration key.
func NewDecider(ctx context.Context, config cfg.Config) (Decider, error) {
	settings := &Settings{}
	if err := config.UnmarshalKey("sampling", settings); err != nil {
		return nil, fmt.Errorf("failed to unmarshal settings: %w", err)
	}

	var ok bool
	var factory StrategyFactory
	var strategy Strategy
	var strategies []Strategy
	var err error

	selectedStrategies := slices.Clone(settings.Strategies)
	if len(selectedStrategies) == 0 {
		selectedStrategies = append(selectedStrategies, "always")
	}

	for _, strategyName := range selectedStrategies {
		if factory, ok = availableStrategies[strategyName]; !ok {
			return nil, fmt.Errorf("unknown strategy %q", strategyName)
		}

		if strategy, err = factory(ctx, config); err != nil {
			return nil, fmt.Errorf("failed to create strategy %q: %w", strategyName, err)
		}

		strategies = append(strategies, strategy)
	}

	return NewDeciderWithInterfaces(strategies, settings, metric.NewWriter()), nil
}

// NewDeciderWithInterfaces creates a new Decider with the given strategies, settings and metric writer.
// This is useful for testing or when manual construction is required.
func NewDeciderWithInterfaces(strategies []Strategy, settings *Settings, writer metric.Writer) Decider {
	return &defaultDecider{
		strategies:   strategies,
		settings:     settings,
		metricWriter: writer,
	}
}

type defaultDecider struct {
	strategies   []Strategy
	settings     *Settings
	metricWriter metric.Writer
}

func (d *defaultDecider) Decide(ctx context.Context, overwriteStrategies ...Strategy) (context.Context, bool, error) {
	if !d.settings.Enabled {
		return ctx, true, nil
	}

	if smplctx.HasSampling(ctx) {
		sampling := smplctx.GetSampling(ctx)

		return ctx, sampling.Sampled, nil
	}

	var err error
	var isApplied, isSampled bool

	strategies := slices.Concat(overwriteStrategies, d.strategies)
	finalIsSampled := true

	for _, strategy := range strategies {
		if isApplied, isSampled, err = strategy(ctx); err != nil {
			return ctx, true, fmt.Errorf("can not apply strategy: %w", err)
		}

		if isApplied {
			finalIsSampled = isSampled

			break
		}
	}

	d.metricWriter.WriteOne(ctx, &metric.Datum{
		Priority:   metric.PriorityLow,
		MetricName: "sampling_decision",
		Unit:       metric.UnitCount,
		Value:      1.0,
		Dimensions: map[string]string{
			"sampled": strconv.FormatBool(finalIsSampled),
		},
	})

	ctx = smplctx.WithSampling(ctx, smplctx.Sampling{Sampled: finalIsSampled})

	return ctx, finalIsSampled, nil
}
