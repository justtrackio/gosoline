package conc_test

import (
	"context"
	"github.com/applike/gosoline/pkg/clock"
	"github.com/applike/gosoline/pkg/conc"
	"github.com/applike/gosoline/pkg/ddb"
	ddbMocks "github.com/applike/gosoline/pkg/ddb/mocks"
	logMocks "github.com/applike/gosoline/pkg/log/mocks"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"testing"
	"time"
)

type DdbLeaderElectionTestCase struct {
	suite.Suite

	logger     *logMocks.Logger
	clock      clock.Clock
	repository *ddbMocks.Repository
	election   *conc.DdbLeaderElection
}

func (s *DdbLeaderElectionTestCase) SetupTest() {
	s.logger = new(logMocks.Logger)
	s.clock = clock.NewFakeClock()
	s.repository = new(ddbMocks.Repository)

	var err error
	s.election, err = conc.NewDdbLeaderElectionWithInterfaces(s.logger, s.clock, s.repository, &conc.DdbLeaderElectionSettings{
		TableName:     "gosoline-leader-election",
		GroupId:       "test",
		LeaseDuration: time.Minute,
	})
	s.NoError(err)
}

func (s *DdbLeaderElectionTestCase) TestSuccess() {
	ctx := context.Background()

	builder := new(ddbMocks.PutItemBuilder)
	builder.On("WithCondition", mock.AnythingOfType("expression.ConditionBuilder")).Return(builder)

	item := &conc.DdbLeaderElectionItem{
		GroupId:      "test",
		MemberId:     "2693674e-66ec-11eb-8dcd-4b6da059a53a",
		LeadingUntil: 449884860,
	}
	result := &ddb.PutItemResult{
		ConditionalCheckFailed: false,
	}

	s.repository.On("PutItemBuilder").Return(builder)
	s.repository.On("PutItem", ctx, builder, item).Return(result, nil)

	isLeader, err := s.election.IsLeader(ctx, "2693674e-66ec-11eb-8dcd-4b6da059a53a")
	s.NoError(err)
	s.True(isLeader)
}

func TestDdbLeaderElectionTestCase(t *testing.T) {
	suite.Run(t, new(DdbLeaderElectionTestCase))
}
