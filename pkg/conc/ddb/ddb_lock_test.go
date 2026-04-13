package ddb_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	clockPkg "github.com/justtrackio/gosoline/pkg/clock"
	"github.com/justtrackio/gosoline/pkg/conc"
	"github.com/justtrackio/gosoline/pkg/conc/ddb"
	"github.com/justtrackio/gosoline/pkg/conc/ddb/mocks"
	logMocks "github.com/justtrackio/gosoline/pkg/log/mocks"
	"github.com/justtrackio/gosoline/pkg/test/matcher"
	"github.com/stretchr/testify/suite"
)

type ddbLockTestSuite struct {
	suite.Suite
	clock       clockPkg.FakeClock
	logger      logMocks.LoggerMock
	ctx         context.Context
	lockManager *mocks.LockManager
	lock        conc.DistributedLock
}

func TestDdbLock(t *testing.T) {
	suite.Run(t, new(ddbLockTestSuite))
}

func (s *ddbLockTestSuite) SetupTest() {
	s.lockManager = mocks.NewLockManager(s.T())
	s.clock = clockPkg.NewFakeClockAt(time.Now())
	s.logger = logMocks.NewLoggerMock(logMocks.WithTestingT(s.T()))
	s.ctx = s.T().Context()
	s.lock = ddb.NewDdbLockFromInterfaces(s.lockManager, s.clock, s.logger, s.ctx, "resource", "token", s.clock.Now().Add(time.Minute))
}

func (s *ddbLockTestSuite) TestRenewLockSuccess() {
	s.lockManager.EXPECT().RenewLock(s.ctx, time.Hour, "resource", "token").Return(s.clock.Now().Add(time.Hour), nil).Once()

	err := s.lock.Renew(s.ctx, time.Hour)
	s.NoError(err)
}

func (s *ddbLockTestSuite) TestRenewLockFails() {
	s.lockManager.EXPECT().RenewLock(s.ctx, time.Hour, "resource", "token").Return(time.Time{}, fmt.Errorf("fail")).Once()

	err := s.lock.Renew(s.ctx, time.Hour)
	s.EqualError(err, "fail")
}

func (s *ddbLockTestSuite) TestReleaseLockSuccess() {
	s.lockManager.EXPECT().ReleaseLock(matcher.Context, "resource", "token").Return(nil).Once()

	err := s.lock.Release()
	s.NoError(err)
}

func (s *ddbLockTestSuite) TestReleaseLockFail() {
	s.lockManager.EXPECT().ReleaseLock(matcher.Context, "resource", "token").Return(fmt.Errorf("fail")).Once()

	err := s.lock.Release()
	s.EqualError(err, "fail")
}

func (s *ddbLockTestSuite) TestReleaseLockDelayedParentCancel() {
	ctx, cancel := context.WithCancel(s.ctx)

	s.lock = ddb.NewDdbLockFromInterfaces(s.lockManager, s.clock, s.logger, ctx, "resource", "token", s.clock.Now().Add(time.Minute))
	cancel()

	s.lockManager.EXPECT().ReleaseLock(matcher.Context, "resource", "token").
		Run(func(ctx context.Context, resource string, token string) {
			s.NoError(ctx.Err())
		}).
		Return(nil).
		Once()

	err := s.lock.Release()
	s.NoError(err)
}

func (s *ddbLockTestSuite) TestReleaseLockCancellationIsIgnored() {
	s.lock = ddb.NewDdbLockFromInterfaces(s.lockManager, s.clock, s.logger, s.ctx, "resource", "token", s.clock.Now().Add(time.Minute))

	s.lockManager.EXPECT().
		ReleaseLock(matcher.Context, "resource", "token").
		Return(context.Canceled).
		Once()

	err := s.lock.Release()
	s.NoError(err)
}

func (s *ddbLockTestSuite) TestReleaseLockDeadlineExceededIsIgnored() {
	s.lock = ddb.NewDdbLockFromInterfaces(s.lockManager, s.clock, s.logger, s.ctx, "resource", "token", s.clock.Now().Add(time.Minute))

	s.lockManager.EXPECT().
		ReleaseLock(matcher.Context, "resource", "token").
		Return(context.DeadlineExceeded).
		Once()

	err := s.lock.Release()
	s.NoError(err)
}
