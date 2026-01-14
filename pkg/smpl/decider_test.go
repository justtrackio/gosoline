package smpl_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/smpl"
	"github.com/justtrackio/gosoline/pkg/smpl/smplctx"
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
	config := cfg.New()
	err := config.Option(cfg.WithConfigMap(map[string]any{
		"sampling.enabled":    true,
		"sampling.strategies": []string{},
	}))
	s.NoError(err)

	decider, err := smpl.NewDecider(context.Background(), config)
	s.NoError(err)
	s.NotNil(decider)

	// Since no strategies provided, it defaults to DecideByAlways
	_, sampled, err := decider.Decide(context.Background())
	s.NoError(err)
	s.True(sampled, "Should default to sampled=true when no strategies configured")
}

func (s *DeciderTestSuite) TestDecide_Disabled() {
	// When enabled=false
	settings := &smpl.Settings{
		Enabled: false,
	}
	// Strategies shouldn't matter if disabled
	neverStrategy, err := smpl.DecideByNever(context.Background(), nil)
	s.Require().NoError(err)

	decider := smpl.NewDeciderWithInterfaces([]smpl.Strategy{neverStrategy}, settings)

	originalCtx := context.Background()
	newCtx, sampled, err := decider.Decide(originalCtx)

	s.NoError(err)
	s.True(sampled, "If disabled, should return sampled=true (meaning 'assumed sampled')")
	s.Equal(originalCtx, newCtx, "If disabled, context should NOT be modified/wrapped")
	s.True(smplctx.IsSampled(newCtx), "Context should act as sampled by default")
}

func (s *DeciderTestSuite) TestDecide_AdditionalStrategiesPrecedence() {
	settings := &smpl.Settings{Enabled: true}
	// Configured strategy says "Always" (Sampled=true)
	alwaysStrategy, err := smpl.DecideByAlways(context.Background(), nil)
	s.Require().NoError(err)

	decider := smpl.NewDeciderWithInterfaces([]smpl.Strategy{alwaysStrategy}, settings)

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
	settings := &smpl.Settings{Enabled: true}

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

	decider := smpl.NewDeciderWithInterfaces([]smpl.Strategy{strat1, strat2, strat3}, settings)

	_, sampled, err := decider.Decide(context.Background())
	s.NoError(err)
	s.False(sampled, "Decision should come from strat2")
	s.Equal([]string{"strat1", "strat2"}, executionOrder, "Should execute strategies in order until one applies")
}

func (s *DeciderTestSuite) TestDecide_ErrorPropagation() {
	settings := &smpl.Settings{Enabled: true}

	errorStrat := func(ctx context.Context) (bool, bool, error) {
		return false, false, fmt.Errorf("strategy failed")
	}

	decider := smpl.NewDeciderWithInterfaces([]smpl.Strategy{errorStrat}, settings)

	_, _, err := decider.Decide(context.Background())
	s.Error(err)
	s.Contains(err.Error(), "can not apply strategy: strategy failed")
}
