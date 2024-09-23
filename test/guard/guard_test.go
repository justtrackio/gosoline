//go:build integration && fixtures

package guard_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/fixtures"
	"github.com/justtrackio/gosoline/pkg/guard"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/test/suite"
	"github.com/selm0/ladon"
)

func fixtureSetsFactory(ctx context.Context, config cfg.Config, logger log.Logger, group string) ([]fixtures.FixtureSet, error) {
	writer, err := fixtures.NewMysqlPlainFixtureWriter(ctx, config, logger, metadata)
	if err != nil {
		return nil, fmt.Errorf("failed to create mysql plain fixture writer: %w", err)
	}

	return []fixtures.FixtureSet{
		fixtures.NewSimpleFixtureSet(namedFixtures, writer),
	}, nil
}

type GuardTestSuite struct {
	suite.Suite
	guard guard.Guard
}

func (s *GuardTestSuite) SetupSuite() []suite.Option {
	return []suite.Option{
		suite.WithLogLevel("debug"),
		suite.WithSharedEnvironment(),
		suite.WithConfigFile("config.dist.yml"),
		suite.WithFixtureSetFactories(fixtureSetsFactory),
	}
}

func (s *GuardTestSuite) SetupTest() error {
	envContext := s.Env().Context()
	config := s.Env().Config()
	logger := s.Env().Logger()

	var err error
	s.guard, err = guard.NewGuard(envContext, config, logger)
	if err != nil {
		return fmt.Errorf("could not create guard: %w", err)
	}

	return nil
}

func (s *GuardTestSuite) TestCrud() {
	pol := ladon.DefaultPolicy{
		ID:          "1",
		Description: "allow all",
		Subjects:    []string{"a:0"},
		Effect:      "allow",
		Resources:   []string{"gsl:a:0:<.+>"},
		Actions:     []string{"<.+>"},
	}

	ctx := context.Background()

	err := s.guard.CreatePolicy(ctx, &pol)
	if !s.NoError(err) {
		return
	}

	pol.Description = "allow everything"

	err = s.guard.UpdatePolicy(ctx, &pol)
	if !s.NoError(err) {
		return
	}

	err = s.guard.DeletePolicy(ctx, &pol)
	s.NoError(err)
}

func (s *GuardTestSuite) TestGetPolicies() {
	ctx := context.Background()

	policies, err := s.guard.GetPolicies(ctx)
	if !s.NoError(err) {
		return
	}

	s.Len(policies, 3)

	policies, err = s.guard.GetPoliciesBySubject(ctx, "r:1")
	if !s.NoError(err) {
		return
	}

	s.Len(policies, 1)
}

func (s *GuardTestSuite) TestIsAllowed() {
	req := ladon.Request{
		Resource: "gsl:e1",
		Action:   "read",
		Subject:  "r:1",
	}

	ctx := context.Background()

	err := s.guard.IsAllowed(ctx, &req)
	s.NoError(err)

	err = s.guard.IsAllowed(ctx, &ladon.Request{})
	s.Error(err, "Request was denied by default")
}

func TestGuardTestSuite(t *testing.T) {
	suite.Run(t, new(GuardTestSuite))
}
