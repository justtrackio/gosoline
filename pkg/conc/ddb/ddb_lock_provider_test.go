package ddb_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/cenkalti/backoff"
	"github.com/justtrackio/gosoline/pkg/clock"
	"github.com/justtrackio/gosoline/pkg/conc"
	concDdb "github.com/justtrackio/gosoline/pkg/conc/ddb"
	"github.com/justtrackio/gosoline/pkg/ddb"
	ddbMocks "github.com/justtrackio/gosoline/pkg/ddb/mocks"
	"github.com/justtrackio/gosoline/pkg/exec"
	logMocks "github.com/justtrackio/gosoline/pkg/log/mocks"
	"github.com/justtrackio/gosoline/pkg/uuid"
	uuidMocks "github.com/justtrackio/gosoline/pkg/uuid/mocks"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

type ddbLockProviderTestSuite struct {
	suite.Suite
	ctx        context.Context
	repo       *ddbMocks.Repository
	clock      clock.Clock
	backOff    backoff.BackOff
	uuidSource *uuidMocks.Uuid
	provider   conc.DistributedLockProvider

	resource string
	token    string
}

func TestDdbLockProvider(t *testing.T) {
	suite.Run(t, new(ddbLockProviderTestSuite))
}

type testBackOff struct {
	backOffs []time.Duration
	index    int
}

func (t *testBackOff) NextBackOff() time.Duration {
	t.index++

	if t.index >= len(t.backOffs) {
		return backoff.Stop
	}

	return t.backOffs[t.index-1]
}

func (t *testBackOff) Reset() {
	t.index = 0
}

func (s *ddbLockProviderTestSuite) SetupSuite() {
	s.resource = fmt.Sprintf("%s-%s", "test", uuid.New().NewV4())
	s.token = uuid.New().NewV4()
}

func (s *ddbLockProviderTestSuite) SetupTest() {
	logger := logMocks.NewLoggerMock(logMocks.WithMockAll, logMocks.WithTestingT(s.T()))
	s.ctx = context.Background()
	s.repo = ddbMocks.NewRepository(s.T())
	s.clock = clock.NewFakeClock()
	s.uuidSource = uuidMocks.NewUuid(s.T())
	s.uuidSource.EXPECT().NewV4().Return(s.token).Once()
	s.backOff = &testBackOff{
		backOffs: []time.Duration{
			time.Millisecond * 1,
			time.Millisecond * 2,
			time.Millisecond * 4,
		},
	}

	s.provider = concDdb.NewDdbLockProviderWithInterfaces(logger, s.repo, s.backOff, s.clock, s.uuidSource, conc.DistributedLockSettings{
		DefaultLockTime: time.Minute,
		Domain:          "test",
	})
}

func (s *ddbLockProviderTestSuite) getAcquireQueryBuilder() *ddbMocks.PutItemBuilder {
	threshold := s.clock.Now().Unix() - 60
	qb := ddbMocks.NewPutItemBuilder(s.T())
	qb.EXPECT().WithCondition(ddb.AttributeNotExists("resource").Or(ddb.Lt("ttl", threshold))).Return(qb)

	s.repo.EXPECT().PutItemBuilder().Return(qb)

	return qb
}

func (s *ddbLockProviderTestSuite) getRenewQueryBuilder() *ddbMocks.UpdateItemBuilder {
	qb := ddbMocks.NewUpdateItemBuilder(s.T())
	qb.EXPECT().WithHash(s.resource).Return(qb).Once()
	qb.EXPECT().WithCondition(ddb.AttributeExists("resource").And(ddb.Eq("token", s.token))).Return(qb).Once()

	s.repo.EXPECT().UpdateItemBuilder().Return(qb).Once()

	return qb
}

func (s *ddbLockProviderTestSuite) getReleaseQueryBuilder(result *ddb.DeleteItemResult, err error) *ddbMocks.DeleteItemBuilder {
	qb := ddbMocks.NewDeleteItemBuilder(s.T())
	qb.EXPECT().WithHash(s.resource).Return(qb).Once()
	qb.EXPECT().WithCondition(ddb.AttributeExists("resource").And(ddb.Eq("token", s.token))).Return(qb).Once()

	s.repo.EXPECT().DeleteItemBuilder().Return(qb).Once()
	s.repo.EXPECT().DeleteItem(mock.AnythingOfType("*exec.stoppableContext"), qb, &concDdb.DdbLockItem{
		Resource: s.resource,
		Token:    s.token,
	}).Return(result, err).Once()

	return qb
}

func (s *ddbLockProviderTestSuite) testAcquireLock(initialLocked bool, initialFail bool) conc.DistributedLock {
	lockItem := &concDdb.DdbLockItem{
		Resource: s.resource,
		Token:    s.token,
		Ttl:      s.clock.Now().Add(time.Minute).Unix(),
	}

	qb := s.getAcquireQueryBuilder()
	if initialLocked {
		s.repo.EXPECT().PutItem(s.ctx, qb, lockItem).
			Return(&ddb.PutItemResult{
				ConditionalCheckFailed: true,
			}, nil).
			Once()
	}
	if initialFail {
		s.repo.EXPECT().PutItem(s.ctx, qb, lockItem).
			Return(nil, errors.New("ddb fails")).
			Once()
	}
	s.repo.EXPECT().PutItem(s.ctx, qb, lockItem).
		Return(&ddb.PutItemResult{}, nil).
		Once()

	l, err := s.provider.Acquire(s.ctx, s.resource[5:])
	s.NotNil(l)
	s.NoError(err)

	return l
}

func (s *ddbLockProviderTestSuite) testAcquireLockIfNotInUse(isInUse bool, err error) (conc.DistributedLock, error) {
	lockItem := &concDdb.DdbLockItem{
		Resource: s.resource,
		Token:    s.token,
		Ttl:      s.clock.Now().Add(time.Minute).Unix(),
	}

	qb := s.getAcquireQueryBuilder()

	switch {
	case err != nil:
		s.repo.EXPECT().PutItem(s.ctx, qb, lockItem).
			Return(nil, err).
			Once()
	case isInUse:
		s.repo.EXPECT().PutItem(s.ctx, qb, lockItem).
			Return(&ddb.PutItemResult{
				ConditionalCheckFailed: true,
			}, nil).
			Once()
	default:
		s.repo.EXPECT().PutItem(s.ctx, qb, lockItem).
			Return(&ddb.PutItemResult{}, nil).
			Once()
	}

	return s.provider.AcquireIfNotInUse(s.ctx, s.resource[5:])
}

func (s *ddbLockProviderTestSuite) TestDdbLockProvider_AcquireCanceled() {
	lockItem := &concDdb.DdbLockItem{
		Resource: s.resource,
		Token:    s.token,
		Ttl:      s.clock.Now().Add(time.Minute).Unix(),
	}

	qb := s.getAcquireQueryBuilder()
	s.repo.EXPECT().PutItem(s.ctx, qb, lockItem).
		Return(nil, exec.RequestCanceledError).
		Once()

	l, err := s.provider.Acquire(s.ctx, s.resource[5:])
	s.Nil(l)
	s.Error(err)
	s.True(exec.IsRequestCanceled(err))

	// we should be able to try to renew and release the lock even if it fails
	// (although that should always fail)
	err = l.Renew(s.ctx, time.Hour)
	s.Error(err)
	s.Equal(conc.ErrNotOwned, err)

	err = l.Release()
	s.Error(err)
	s.Equal(conc.ErrNotOwned, err)
}

func (s *ddbLockProviderTestSuite) TestDdbLockProvider_AcquireFails() {
	lockItem := &concDdb.DdbLockItem{
		Resource: s.resource,
		Token:    s.token,
		Ttl:      s.clock.Now().Add(time.Minute).Unix(),
	}

	qb := s.getAcquireQueryBuilder()
	// call PutItem at least 3 times - we might call it more often (up to 4 times),
	// but there is no way to encode that information with the mock library
	s.repo.EXPECT().PutItem(s.ctx, qb, lockItem).Return(&ddb.PutItemResult{
		ConditionalCheckFailed: true,
	}, nil).Twice()
	s.repo.EXPECT().PutItem(s.ctx, qb, lockItem).Return(&ddb.PutItemResult{
		ConditionalCheckFailed: true,
	}, nil).Once()

	l, err := s.provider.Acquire(s.ctx, s.resource[5:])
	s.Nil(l)
	s.Error(err)
	s.Equal(conc.ErrOwnedLock, err)
}

func (s *ddbLockProviderTestSuite) TestDdbLockProvider_AcquireFailsThenSucceeds() {
	_ = s.testAcquireLock(true, true)
}

func (s *ddbLockProviderTestSuite) TestDdbLockProvider_AcquireThenReleaseTooLate() {
	l := s.testAcquireLock(false, false)
	_ = s.getReleaseQueryBuilder(&ddb.DeleteItemResult{
		ConditionalCheckFailed: true,
	}, nil)

	err := l.Release()
	s.Error(err)
	s.Equal(conc.ErrNotOwned, err)
}

func (s *ddbLockProviderTestSuite) TestDdbLockProvider_AcquireThenReleaseError() {
	l := s.testAcquireLock(false, true)
	dbErr := errors.New("db error")
	_ = s.getReleaseQueryBuilder(nil, dbErr)

	err := l.Release()
	s.Error(err)
	s.Equal(dbErr, err)
}

func (s *ddbLockProviderTestSuite) TestDdbLockProvider_AcquireOnRetryThenRelease() {
	l := s.testAcquireLock(true, false)
	_ = s.getReleaseQueryBuilder(&ddb.DeleteItemResult{
		ConditionalCheckFailed: false,
	}, nil)

	err := l.Release()
	s.NoError(err)
}

func (s *ddbLockProviderTestSuite) TestDdbLockProvider_AcquireThenRelease() {
	l := s.testAcquireLock(false, false)
	_ = s.getReleaseQueryBuilder(&ddb.DeleteItemResult{
		ConditionalCheckFailed: false,
	}, nil)

	err := l.Release()
	s.NoError(err)
}

func (s *ddbLockProviderTestSuite) TestDdbLockProvider_AcquireThenRenewCanceled() {
	l := s.testAcquireLock(true, false)
	qb := s.getRenewQueryBuilder()
	lockItem := &concDdb.DdbLockItem{
		Resource: s.resource,
		Token:    s.token,
		Ttl:      s.clock.Now().Add(time.Hour).Unix(),
	}

	s.repo.EXPECT().UpdateItem(s.ctx, qb, lockItem).
		Return(nil, exec.RequestCanceledError).
		Once()

	err := l.Renew(s.ctx, time.Hour)
	s.Error(err)
	s.True(exec.IsRequestCanceled(err))
}

func (s *ddbLockProviderTestSuite) TestDdbLockProvider_AcquireThenRenewFails() {
	l := s.testAcquireLock(true, false)
	qb := s.getRenewQueryBuilder()
	lockItem := &concDdb.DdbLockItem{
		Resource: s.resource,
		Token:    s.token,
		Ttl:      s.clock.Now().Add(time.Hour).Unix(),
	}

	s.repo.EXPECT().UpdateItem(s.ctx, qb, lockItem).Return(&ddb.UpdateItemResult{
		ConditionalCheckFailed: true,
	}, nil).Once()

	err := l.Renew(s.ctx, time.Hour)
	s.Error(err)
	s.Equal(conc.ErrNotOwned, err)
}

func (s *ddbLockProviderTestSuite) TestDdbLockProvider_AcquireThenRenewErrorsAndSucceeds() {
	l := s.testAcquireLock(true, false)
	qb1 := s.getRenewQueryBuilder()
	qb2 := s.getRenewQueryBuilder()
	lockItem := &concDdb.DdbLockItem{
		Resource: s.resource,
		Token:    s.token,
		Ttl:      s.clock.Now().Add(time.Hour).Unix(),
	}

	s.repo.EXPECT().UpdateItem(s.ctx, qb1, lockItem).Return(nil, errors.New("db error")).Once()
	s.repo.EXPECT().UpdateItem(s.ctx, qb2, lockItem).Return(&ddb.UpdateItemResult{}, nil).Once()

	err := l.Renew(s.ctx, time.Hour)
	s.NoError(err)
}

func (s *ddbLockProviderTestSuite) TestDdbLockProvider_AcquireIfAlreadyInUse_AcquireThenRenewAndRelease() {
	l, err := s.testAcquireLockIfNotInUse(false, nil)
	s.NotNil(l)
	s.NoError(err)

	qb := s.getRenewQueryBuilder()
	renewedLockItem := &concDdb.DdbLockItem{
		Resource: s.resource,
		Token:    s.token,
		Ttl:      s.clock.Now().Add(time.Hour).Unix(),
	}

	s.repo.EXPECT().UpdateItem(s.ctx, qb, renewedLockItem).Return(&ddb.UpdateItemResult{
		ConditionalCheckFailed: false,
	}, nil).Once()

	_ = s.getReleaseQueryBuilder(&ddb.DeleteItemResult{
		ConditionalCheckFailed: false,
	}, nil)

	err = l.Renew(s.ctx, time.Hour)
	s.NoError(err)

	err = l.Release()
	s.NoError(err)
}

func (s *ddbLockProviderTestSuite) TestDdbLockProvider_AcquireIfAlreadyInUse_IsInUse() {
	l, err := s.testAcquireLockIfNotInUse(true, nil)

	s.Nil(l)
	s.NoError(err)
}

func (s *ddbLockProviderTestSuite) TestDdbLockProvider_AcquireIfAlreadyInUse_Error() {
	l, err := s.testAcquireLockIfNotInUse(false, errors.New("ddb fails"))

	s.Nil(l)
	s.EqualError(err, "ddb fails")
}
