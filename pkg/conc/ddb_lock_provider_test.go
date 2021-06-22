package conc_test

import (
	"context"
	"errors"
	"fmt"
	"github.com/applike/gosoline/pkg/clock"
	"github.com/applike/gosoline/pkg/conc"
	"github.com/applike/gosoline/pkg/ddb"
	ddbMocks "github.com/applike/gosoline/pkg/ddb/mocks"
	"github.com/applike/gosoline/pkg/exec"
	logMocks "github.com/applike/gosoline/pkg/log/mocks"
	"github.com/applike/gosoline/pkg/uuid"
	uuidMocks "github.com/applike/gosoline/pkg/uuid/mocks"
	"github.com/cenkalti/backoff"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"testing"
	"time"
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
	logger := logMocks.NewLoggerMockedAll()
	s.ctx = context.Background()
	s.repo = new(ddbMocks.Repository)
	s.clock = clock.NewFakeClock()
	s.uuidSource = new(uuidMocks.Uuid)
	s.uuidSource.On("NewV4").Return(s.token).Once()
	s.backOff = &testBackOff{
		backOffs: []time.Duration{
			time.Millisecond * 1,
			time.Millisecond * 2,
			time.Millisecond * 4,
		},
	}

	s.provider = conc.NewDdbLockProviderWithInterfaces(logger, s.repo, s.backOff, s.clock, s.uuidSource, conc.DistributedLockSettings{
		DefaultLockTime: time.Minute,
		Domain:          "test",
	})
}

func (s *ddbLockProviderTestSuite) getAcquireQueryBuilder() *ddbMocks.PutItemBuilder {
	threshold := s.clock.Now().Unix() - 60
	qb := new(ddbMocks.PutItemBuilder)
	qb.On("WithCondition", ddb.AttributeNotExists("resource").Or(ddb.Lt("ttl", threshold))).Return(qb)

	s.repo.On("PutItemBuilder").Return(qb)

	return qb
}

func (s *ddbLockProviderTestSuite) getRenewQueryBuilder() *ddbMocks.UpdateItemBuilder {
	qb := new(ddbMocks.UpdateItemBuilder)
	qb.On("WithHash", s.resource).Return(qb)
	qb.On("WithCondition", ddb.AttributeExists("resource").And(ddb.Eq("token", s.token))).Return(qb)

	s.repo.On("UpdateItemBuilder").Return(qb).Once()

	return qb
}

func (s *ddbLockProviderTestSuite) getReleaseQueryBuilder(result *ddb.DeleteItemResult, err error) *ddbMocks.DeleteItemBuilder {
	qb := new(ddbMocks.DeleteItemBuilder)
	qb.On("WithHash", s.resource).Return(qb).Once()
	qb.On("WithCondition", ddb.AttributeExists("resource").And(ddb.Eq("token", s.token))).Return(qb).Once()

	s.repo.On("DeleteItemBuilder").Return(qb).Once()
	s.repo.On("DeleteItem", mock.AnythingOfType("*exec.delayedCancelContext"), qb, &conc.DdbLockItem{
		Resource: s.resource,
		Token:    s.token,
	}).Return(result, err)

	return qb
}

func (s *ddbLockProviderTestSuite) testAcquireLock(initialLocked bool, initialFail bool) conc.DistributedLock {
	lockItem := &conc.DdbLockItem{
		Resource: s.resource,
		Token:    s.token,
		Ttl:      s.clock.Now().Add(time.Minute).Unix(),
	}

	qb := s.getAcquireQueryBuilder()
	if initialLocked {
		s.repo.On("PutItem", s.ctx, qb, lockItem).
			Return(&ddb.PutItemResult{
				ConditionalCheckFailed: true,
			}, nil).
			Once()
	}
	if initialFail {
		s.repo.On("PutItem", s.ctx, qb, lockItem).
			Return(nil, errors.New("ddb fails")).
			Once()
	}
	s.repo.On("PutItem", s.ctx, qb, lockItem).
		Return(&ddb.PutItemResult{}, nil).
		Once()

	l, err := s.provider.Acquire(s.ctx, s.resource[5:])
	s.NotNil(l)
	s.NoError(err)
	qb.AssertExpectations(s.T())

	return l
}

func (s *ddbLockProviderTestSuite) TestDdbLockProvider_AcquireCanceled() {
	lockItem := &conc.DdbLockItem{
		Resource: s.resource,
		Token:    s.token,
		Ttl:      s.clock.Now().Add(time.Minute).Unix(),
	}

	qb := s.getAcquireQueryBuilder()
	s.repo.On("PutItem", s.ctx, qb, lockItem).
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

	qb.AssertExpectations(s.T())
	s.repo.AssertExpectations(s.T())
	s.uuidSource.AssertExpectations(s.T())
}

func (s *ddbLockProviderTestSuite) TestDdbLockProvider_AcquireFails() {
	lockItem := &conc.DdbLockItem{
		Resource: s.resource,
		Token:    s.token,
		Ttl:      s.clock.Now().Add(time.Minute).Unix(),
	}

	qb := s.getAcquireQueryBuilder()
	// call PutItem at least 3 times - we might call it more often (up to 4 times),
	// but there is no way to encode that information with the mock library
	s.repo.On("PutItem", s.ctx, qb, lockItem).Return(&ddb.PutItemResult{
		ConditionalCheckFailed: true,
	}, nil).Twice()
	s.repo.On("PutItem", s.ctx, qb, lockItem).Return(&ddb.PutItemResult{
		ConditionalCheckFailed: true,
	}, nil)

	l, err := s.provider.Acquire(s.ctx, s.resource[5:])
	s.Nil(l)
	s.Error(err)
	s.Equal(conc.ErrOwnedLock, err)

	qb.AssertExpectations(s.T())
	s.repo.AssertExpectations(s.T())
	s.uuidSource.AssertExpectations(s.T())
}

func (s *ddbLockProviderTestSuite) TestDdbLockProvider_AcquireFailsThenSucceeds() {
	_ = s.testAcquireLock(true, true)

	s.repo.AssertExpectations(s.T())
	s.uuidSource.AssertExpectations(s.T())
}

func (s *ddbLockProviderTestSuite) TestDdbLockProvider_AcquireThenReleaseTooLate() {
	l := s.testAcquireLock(false, false)
	qb := s.getReleaseQueryBuilder(&ddb.DeleteItemResult{
		ConditionalCheckFailed: true,
	}, nil)

	err := l.Release()
	s.Error(err)
	s.Equal(conc.ErrNotOwned, err)

	qb.AssertExpectations(s.T())
	s.repo.AssertExpectations(s.T())
	s.uuidSource.AssertExpectations(s.T())
}

func (s *ddbLockProviderTestSuite) TestDdbLockProvider_AcquireThenReleaseError() {
	l := s.testAcquireLock(false, true)
	dbErr := errors.New("db error")
	qb := s.getReleaseQueryBuilder(nil, dbErr)

	err := l.Release()
	s.Error(err)
	s.Equal(dbErr, err)

	qb.AssertExpectations(s.T())
	s.repo.AssertExpectations(s.T())
	s.uuidSource.AssertExpectations(s.T())
}

func (s *ddbLockProviderTestSuite) TestDdbLockProvider_AcquireThenRelease() {
	l := s.testAcquireLock(true, false)
	qb := s.getReleaseQueryBuilder(&ddb.DeleteItemResult{
		ConditionalCheckFailed: false,
	}, nil)

	err := l.Release()
	s.NoError(err)

	qb.AssertExpectations(s.T())
	s.repo.AssertExpectations(s.T())
	s.uuidSource.AssertExpectations(s.T())
}

func (s *ddbLockProviderTestSuite) TestDdbLockProvider_AcquireThenRenewCanceled() {
	l := s.testAcquireLock(true, false)
	qb := s.getRenewQueryBuilder()
	lockItem := &conc.DdbLockItem{
		Resource: s.resource,
		Token:    s.token,
		Ttl:      s.clock.Now().Add(time.Hour).Unix(),
	}

	s.repo.On("UpdateItem", s.ctx, qb, lockItem).
		Return(nil, exec.RequestCanceledError).
		Once()

	err := l.Renew(s.ctx, time.Hour)
	s.Error(err)
	s.True(exec.IsRequestCanceled(err))

	qb.AssertExpectations(s.T())
	s.repo.AssertExpectations(s.T())
	s.uuidSource.AssertExpectations(s.T())
}

func (s *ddbLockProviderTestSuite) TestDdbLockProvider_AcquireThenRenewFails() {
	l := s.testAcquireLock(true, false)
	qb := s.getRenewQueryBuilder()
	lockItem := &conc.DdbLockItem{
		Resource: s.resource,
		Token:    s.token,
		Ttl:      s.clock.Now().Add(time.Hour).Unix(),
	}

	s.repo.On("UpdateItem", s.ctx, qb, lockItem).Return(&ddb.UpdateItemResult{
		ConditionalCheckFailed: true,
	}, nil).Once()

	err := l.Renew(s.ctx, time.Hour)
	s.Error(err)
	s.Equal(conc.ErrNotOwned, err)

	qb.AssertExpectations(s.T())
	s.repo.AssertExpectations(s.T())
	s.uuidSource.AssertExpectations(s.T())
}

func (s *ddbLockProviderTestSuite) TestDdbLockProvider_AcquireThenRenewErrorsAndSucceeds() {
	l := s.testAcquireLock(true, false)
	qb1 := s.getRenewQueryBuilder()
	qb2 := s.getRenewQueryBuilder()
	lockItem := &conc.DdbLockItem{
		Resource: s.resource,
		Token:    s.token,
		Ttl:      s.clock.Now().Add(time.Hour).Unix(),
	}

	s.repo.On("UpdateItem", s.ctx, qb1, lockItem).Return(nil, errors.New("db error")).Once()
	s.repo.On("UpdateItem", s.ctx, qb2, lockItem).Return(&ddb.UpdateItemResult{}, nil).Once()

	err := l.Renew(s.ctx, time.Hour)
	s.NoError(err)

	qb1.AssertExpectations(s.T())
	qb2.AssertExpectations(s.T())
	s.repo.AssertExpectations(s.T())
	s.uuidSource.AssertExpectations(s.T())
}

func TestDdbLockProvider(t *testing.T) {
	suite.Run(t, new(ddbLockProviderTestSuite))
}
