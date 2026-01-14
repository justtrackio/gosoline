package smpl_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/metric"
	metricMocks "github.com/justtrackio/gosoline/pkg/metric/mocks"
	"github.com/justtrackio/gosoline/pkg/smpl"
	"github.com/justtrackio/gosoline/pkg/smpl/smplctx"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

type DeciderTestSuite struct {
	suite.Suite
}

func TestDeciderTestSuite(t *testing.T) {
	suite.Run(t, new(DeciderTestSuite))
}

func (s *DeciderTestSuite) TestNewDecider_UnknownStrategy() {
	config := cfg.New()
	err := config.Option(cfg.WithConfigMap(map[string]any{
		"sampling.enabled":    true,
		"sampling.strategies": []string{"unknown-strategy-name"},
	}))
	s.NoError(err)

	decider, err := smpl.NewDecider(context.Background(), config)
	s.Error(err)
	s.Contains(err.Error(), "unknown strategy \"unknown-strategy-name\"")
	s.Nil(decider)
}

func (s *DeciderTestSuite) TestNewDecider_DefaultWhenStrategiesEmpty() {
	decider, writer := s.newDecider(true)
	s.expectSamplingMetric(writer, true)

	// Since no strategies provided, it defaults to DecideByAlways
	_, sampled, err := decider.Decide(context.Background())
	s.NoError(err)
	s.True(sampled, "Should default to sampled=true when no strategies configured")
}

func (s *DeciderTestSuite) TestDecide_Disabled() {
	// When enabled=false
	// Strategies shouldn't matter if disabled
	neverStrategy, err := smpl.DecideByNever(context.Background(), nil)
	s.Require().NoError(err)

	decider, _ := s.newDecider(false, neverStrategy)

	originalCtx := context.Background()
	newCtx, sampled, err := decider.Decide(originalCtx)

	s.NoError(err)
	s.True(sampled, "If disabled, should return sampled=true (meaning 'assumed sampled')")
	s.Equal(originalCtx, newCtx, "If disabled, context should NOT be modified/wrapped")
	s.True(smplctx.IsSampled(newCtx), "Context should act as sampled by default")
}

func (s *DeciderTestSuite) TestDecide_AdditionalStrategiesPrecedence() {
	// Configured strategy says "Always" (Sampled=true)
	alwaysStrategy, err := smpl.DecideByAlways(context.Background(), nil)
	s.Require().NoError(err)

	decider, writer := s.newDecider(true, alwaysStrategy)
	s.expectSamplingMetric(writer, false)

	// Additional strategy says "Never" (Sampled=false)
	additional := func(ctx context.Context) (bool, bool, error) {
		return true, false, nil // applied=true, sampled=false
	}

	newCtx, sampled, err := decider.Decide(context.Background(), additional)
	s.NoError(err)
	s.False(sampled, "Additional strategy should take precedence and override configured ones")
	s.False(smplctx.IsSampled(newCtx), "Context should reflect the decision")
}

func (s *DeciderTestSuite) TestDecide_FirstAppliedWins() {
	var executionOrder []string

	strat1 := func(ctx context.Context) (bool, bool, error) {
		executionOrder = append(executionOrder, "strat1")

		return false, false, nil
	}
	strat2 := func(ctx context.Context) (bool, bool, error) {
		executionOrder = append(executionOrder, "strat2")

		return true, false, nil
	}
	strat3 := func(ctx context.Context) (bool, bool, error) {
		executionOrder = append(executionOrder, "strat3")

		return true, true, nil
	}

	decider, writer := s.newDecider(true, strat1, strat2, strat3)
	s.expectSamplingMetric(writer, false)

	_, sampled, err := decider.Decide(context.Background())
	s.NoError(err)
	s.False(sampled, "Decision should come from strat2")
	s.Equal([]string{"strat1", "strat2"}, executionOrder, "Should execute strategies in order until one applies")
}

func (s *DeciderTestSuite) TestDecide_ErrorPropagation() {
	errorStrat := func(ctx context.Context) (bool, bool, error) {
		return false, false, fmt.Errorf("strategy failed")
	}

	decider, _ := s.newDecider(true, errorStrat)

	_, _, err := decider.Decide(context.Background())
	s.Error(err)
	s.Contains(err.Error(), "can not apply strategy: strategy failed")
}

func (s *DeciderTestSuite) TestDecideCases() {
	always, err := smpl.DecideByAlways(context.Background(), nil)
	s.NoError(err)
	never, err := smpl.DecideByNever(context.Background(), nil)
	s.NoError(err)
	errorStrat := func(ctx context.Context) (bool, bool, error) {
		return false, false, fmt.Errorf("fail")
	}

	tests := []struct {
		name          string
		strategies    []smpl.Strategy
		setupContext  func(ctx context.Context) context.Context
		expectMetric  bool
		expectSampled bool
		expectError   bool
	}{
		{
			name:          "decided true -> emit metric",
			strategies:    []smpl.Strategy{always},
			expectMetric:  true,
			expectSampled: true,
		},
		{
			name:          "decided false -> emit metric",
			strategies:    []smpl.Strategy{never},
			expectMetric:  true,
			expectSampled: false,
		},
		{
			name:       "already decided -> no metric",
			strategies: []smpl.Strategy{always},
			setupContext: func(ctx context.Context) context.Context {
				return smplctx.WithSampling(ctx, smplctx.Sampling{Sampled: true})
			},
			expectMetric: false,
		},
		{
			name:        "error -> no metric",
			strategies:  []smpl.Strategy{errorStrat},
			expectError: true,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			decider, writer := s.newDecider(true, tt.strategies...)

			if tt.expectMetric {
				s.expectSamplingMetric(writer, tt.expectSampled)
			} else {
				writer.AssertNotCalled(s.T(), "WriteOne", mock.Anything, mock.Anything)
			}

			ctx := context.Background()
			if tt.setupContext != nil {
				ctx = tt.setupContext(ctx)
			}

			_, _, err := decider.Decide(ctx)

			if tt.expectError {
				s.Error(err)
			} else {
				s.NoError(err)
			}
		})
	}
}

func (s *DeciderTestSuite) newDecider(enabled bool, strategies ...smpl.Strategy) (smpl.Decider, *metricMocks.Writer) {
	settings := &smpl.Settings{
		Enabled: enabled,
	}
	writer := metricMocks.NewWriter(s.T())
	decider := smpl.NewDeciderWithInterfaces(strategies, settings, writer)

	return decider, writer
}

func (s *DeciderTestSuite) expectSamplingMetric(writer *metricMocks.Writer, sampled bool) {
	sampledStr := "false"
	if sampled {
		sampledStr = "true"
	}

	writer.EXPECT().WriteOne(mock.Anything, mock.MatchedBy(func(d *metric.Datum) bool {
		return d.Priority == metric.PriorityHigh &&
			d.MetricName == "sampling_decision" &&
			d.Unit == metric.UnitCount &&
			d.Value == 1.0 &&
			d.Dimensions["sampled"] == sampledStr
	})).Return()
}
