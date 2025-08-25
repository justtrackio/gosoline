//go:build integration && fixtures

package output

import (
	"testing"

	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/test/suite"
)

type OutputTestSuite struct {
	suite.Suite
}

func TestOutputTestSuite(t *testing.T) {
	suite.Run(t, new(OutputTestSuite))
}

func (s *OutputTestSuite) SetupSuite() []suite.Option {
	return []suite.Option{
		suite.WithLogLevel(log.LevelDebug),
		suite.WithConfigFile("config.dist.yml"),
		suite.WithModule("output-module", NewOutputModule),
	}
}

func (s *OutputTestSuite) TestWrite(app suite.AppUnderTest) {
	app.WaitDone()

	clt := s.Env().Kafka("default").Client()

	offsets, err := clt.ListEndOffsets(s.Env().Context(), "gosoline-test-test-grp-testEvent")
	s.NoError(err)

	offset, ok := offsets.Lookup("gosoline-test-test-grp-testEvent", 0)
	s.True(ok)

	s.Equal(int64(1), offset.Offset)
}
