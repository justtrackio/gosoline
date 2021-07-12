// +build integration,fixtures

package guard_test

import (
	"fmt"
	"testing"

	"github.com/ory/ladon"

	"github.com/applike/gosoline/pkg/guard"

	"github.com/applike/gosoline/pkg/test/suite"
)

type GuardTestSuite struct {
	suite.Suite
	guard guard.Guard
}

func (s *GuardTestSuite) SetupSuite() []suite.Option {
	return []suite.Option{
		suite.WithLogLevel("debug"),
		suite.WithSharedEnvironment(),
		suite.WithConfigFile("config.dist.yml"),
		suite.WithFixtures(buildFixtures()),
	}
}

func (s *GuardTestSuite) SetupTest() error {
	logger := s.Env().Logger()

	var err error
	s.guard, err = guard.NewGuard(s.Env().Config(), logger)
	if err != nil {
		return fmt.Errorf("could not create guard: %w", err)
	}

	return nil
}

func (s *GuardTestSuite) TestCrud(app suite.AppUnderTest) {
	pol := ladon.DefaultPolicy{
		ID:          "1",
		Description: "allow all",
		Subjects:    []string{"a:0"},
		Effect:      "allow",
		Resources:   []string{"gsl:a:0:<.+>"},
		Actions:     []string{"<.+>"},
	}

	err := s.guard.CreatePolicy(&pol)
	if !s.NoError(err) {
		return
	}

	pol.Description = "allow everything"

	err = s.guard.UpdatePolicy(&pol)
	if !s.NoError(err) {
		return
	}

	err = s.guard.DeletePolicy(&pol)
	s.NoError(err)
}

func (s *GuardTestSuite) TestGetPolicies(app suite.AppUnderTest) {
	policies, err := s.guard.GetPolicies()
	if !s.NoError(err) {
		return
	}

	s.Len(policies, 2)

	policies, err = s.guard.GetPoliciesBySubject("r:1")
	if !s.NoError(err) {
		return
	}

	s.Len(policies, 1)
}

func (s *GuardTestSuite) TestIsAllowed(app suite.AppUnderTest) {
	req := ladon.Request{
		Resource: "gsl:e1",
		Action:   "read",
		Subject:  "r:1",
	}

	err := s.guard.IsAllowed(&req)
	s.NoError(err)

	err = s.guard.IsAllowed(&ladon.Request{})
	s.Error(err, "Request was denied by default")
}

func TestGuardTestSuite(t *testing.T) {
	suite.Run(t, new(GuardTestSuite))
}
