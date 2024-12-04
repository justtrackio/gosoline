package suite_test

import (
	"context"
	"testing"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/kernel"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/test/suite"
	"github.com/stretchr/testify/assert"
)

type ApplicationTestSuite struct {
	kernel.EssentialModule
	suite.Suite
	called bool
}

func TestApplicationTestSuite(t *testing.T) {
	var s ApplicationTestSuite
	suite.Run(t, &s)
	assert.True(t, s.called)
}

func (s *ApplicationTestSuite) SetupSuite() []suite.Option {
	return []suite.Option{
		suite.WithLogLevel("info"),
		suite.WithModule("testModule", func(ctx context.Context, config cfg.Config, logger log.Logger) (kernel.Module, error) {
			return s, nil
		}),
		suite.WithSharedEnvironment(),
	}
}

func (s *ApplicationTestSuite) Run(_ context.Context) error {
	return nil
}

func (s *ApplicationTestSuite) TestApp(app suite.AppUnderTest) {
	defer app.WaitDone()
	defer app.Stop()

	s.called = true
}
