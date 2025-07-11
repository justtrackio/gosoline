package ddb_test

import (
	"testing"
	"time"

	"github.com/justtrackio/gosoline/pkg/clock"
	concDdb "github.com/justtrackio/gosoline/pkg/conc/ddb"
	"github.com/justtrackio/gosoline/pkg/ddb"
	ddbMocks "github.com/justtrackio/gosoline/pkg/ddb/mocks"
	logMocks "github.com/justtrackio/gosoline/pkg/log/mocks"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

type DdbLeaderElectionTestCase struct {
	suite.Suite

	logger     *logMocks.Logger
	clock      clock.Clock
	repository *ddbMocks.Repository
	election   *concDdb.DdbLeaderElection
}

func (s *DdbLeaderElectionTestCase) SetupTest() {
	s.logger = logMocks.NewLogger(s.T())
	s.clock = clock.NewFakeClock()
	s.repository = ddbMocks.NewRepository(s.T())

	var err error
	s.election, err = concDdb.NewDdbLeaderElectionWithInterfaces(s.logger, s.clock, s.repository, &concDdb.DdbLeaderElectionSettings{
		Naming: concDdb.TableNamingSettings{
			Pattern: "gosoline-leader-election",
		},
		GroupId:       "test",
		LeaseDuration: time.Minute,
	})
	s.NoError(err)
}

func (s *DdbLeaderElectionTestCase) TestSuccess() {
	ctx := s.T().Context()

	builder := ddbMocks.NewPutItemBuilder(s.T())
	builder.EXPECT().WithCondition(mock.AnythingOfType("expression.ConditionBuilder")).Return(builder)

	item := &concDdb.DdbLeaderElectionItem{
		GroupId:      "test",
		MemberId:     "2693674e-66ec-11eb-8dcd-4b6da059a53a",
		LeadingUntil: 449884860,
	}
	result := &ddb.PutItemResult{
		ConditionalCheckFailed: false,
	}

	s.repository.EXPECT().PutItemBuilder().Return(builder)
	s.repository.EXPECT().PutItem(ctx, builder, item).Return(result, nil)

	isLeader, err := s.election.IsLeader(ctx, "2693674e-66ec-11eb-8dcd-4b6da059a53a")
	s.NoError(err)
	s.True(isLeader)
}

func TestDdbLeaderElectionTestCase(t *testing.T) {
	suite.Run(t, new(DdbLeaderElectionTestCase))
}
