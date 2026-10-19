//go:build integration && fixtures

package limit

import (
	"context"
	"testing"
	"time"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/clock"
	"github.com/justtrackio/gosoline/pkg/ddb"
	"github.com/justtrackio/gosoline/pkg/kernel"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/mdl"
	"github.com/justtrackio/gosoline/pkg/test/suite"
)

type fixedWindowDdbTestSuite struct {
	suite.Suite
	backend *fixedWindowDdb
	clock   clock.FakeClock
}

func TestFixedWindowDdb(t *testing.T) {
	suite.Run(t, new(fixedWindowDdbTestSuite))
}

func (s *fixedWindowDdbTestSuite) SetupSuite() []suite.Option {
	return []suite.Option{
		suite.WithConfigMap(map[string]any{
			"app.namespace":               "{app.env}.{app.name}",
			"app.model_id.domain_pattern": "{app.env}.{app.name}",
		}),
		suite.WithModule("limiter", s.newLimiterModule),
	}
}

func (s *fixedWindowDdbTestSuite) newLimiterModule(ctx context.Context, config cfg.Config, logger log.Logger) (kernel.Module, error) {
	limiter, err := NewFixedWindowDdb(ctx, config, logger, &ddb.Settings{
		ModelId: mdl.ModelId{Name: "rate-limits"},
		Main:    ddb.MainSettings{ReadCapacityUnits: 1, WriteCapacityUnits: 1},
	}, FixedWindowConfig{Name: "expiry-test", Cap: 1, Window: 10 * time.Second})
	if err != nil {
		return nil, err
	}
	s.clock = clock.NewFakeClockAt(time.Now().UTC().Add(time.Hour).Truncate(time.Second))
	s.backend = limiter.(*fixedWindow).backend.(*fixedWindowDdb)
	s.backend.clock = s.clock

	return kernel.NewModuleFunc(func(context.Context) error { return nil }), nil
}

func (s *fixedWindowDdbTestSuite) TestRenewsAtExpiry(app suite.AppUnderTest) {
	app.WaitDone()

	for _, step := range []struct {
		advance time.Duration
		count   int
		ttl     time.Duration
	}{
		{count: 1, ttl: 10 * time.Second},
		{advance: 9 * time.Second, count: 2, ttl: time.Second},
		{advance: time.Second, count: 1, ttl: 10 * time.Second},
		{advance: 10 * time.Second, count: 1, ttl: 10 * time.Second},
	} {
		s.clock.Advance(step.advance)
		count, ttl, err := s.backend.Increment(s.Env().Context(), "network")
		s.Require().NoError(err)
		s.Require().Equal(step.count, *count, "counter at %s", s.clock.Now())
		s.Require().Equal(step.ttl, *ttl, "TTL at %s", s.clock.Now())
	}
}
