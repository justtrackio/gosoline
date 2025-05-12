package db_repo_test

import (
	"context"
	"errors"
	"testing"

	"github.com/go-sql-driver/mysql"
	dbRepo "github.com/justtrackio/gosoline/pkg/db-repo"
	dbMocks "github.com/justtrackio/gosoline/pkg/db-repo/mocks"
	loggerMocks "github.com/justtrackio/gosoline/pkg/log/mocks"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

type retryingRepositoryTestSuite struct {
	suite.Suite

	ctx          context.Context
	logger       *loggerMocks.Logger
	model        *dbMocks.ModelBased
	repo         *dbMocks.Repository
	retryingRepo *dbRepo.RetryingRepository
}

func (s *retryingRepositoryTestSuite) SetupTest() {
	s.ctx = context.Background()
	s.logger = &loggerMocks.Logger{}
	s.model = &dbMocks.ModelBased{}
	s.repo = dbMocks.NewRepository(s.T())
	s.retryingRepo = dbRepo.NewRetryingRepository(s.logger, s.repo, dbRepo.DefaultRetryEvalFunc, dbRepo.DefaultBackoffConfig())
}

func (s *retryingRepositoryTestSuite) TestCreate_Success() {
	s.repo.EXPECT().Create(mock.AnythingOfType("context.backgroundCtx"), s.model).Return(nil).Once()

	err := s.retryingRepo.Create(s.ctx, s.model)

	s.NoError(err)
	s.repo.AssertExpectations(s.T())
}

func (s *retryingRepositoryTestSuite) TestUpdate_NonRetryable() {
	errPermanent := errors.New("non-retryable update error")
	s.repo.EXPECT().Update(mock.AnythingOfType("context.backgroundCtx"), s.model).Return(errPermanent).Once()
	s.logger.EXPECT().Warn("non-retryable DB error: %v", errPermanent)

	err := s.retryingRepo.Update(s.ctx, s.model)

	s.EqualError(err, "non-retryable update error")
	s.repo.AssertExpectations(s.T())
}

func (s *retryingRepositoryTestSuite) TestUpdate_WithRetry() {
	attempts := 0
	errDeadlock := &mysql.MySQLError{Number: dbRepo.DeadlockErrorCode, Message: "Deadlock found when trying to get lock"}
	s.repo.On("Update", mock.Anything, s.model).Return(func(ctx context.Context, m dbRepo.ModelBased) error {
		attempts++
		if attempts < 3 {
			return errDeadlock
		}

		return nil
	}).Times(3)
	s.logger.EXPECT().Warn("deadlock detected, will retry: %v", errDeadlock.Message)

	err := s.retryingRepo.Update(s.ctx, s.model)

	s.NoError(err)
	s.repo.AssertExpectations(s.T())
	s.Equal(3, attempts)
}

func (s *retryingRepositoryTestSuite) TestUpdate_MaxRetriesExceeded() {
	attempts := 0
	maxRetries := 5

	errDeadlock := &mysql.MySQLError{Number: dbRepo.DeadlockErrorCode, Message: "Deadlock found"}

	s.retryingRepo = dbRepo.NewRetryingRepository(s.logger, s.repo, dbRepo.DefaultRetryEvalFunc, dbRepo.DefaultBackoffConfig())

	s.repo.On("Update", mock.Anything, s.model).Return(func(ctx context.Context, m dbRepo.ModelBased) error {
		attempts++

		return errDeadlock
	}).Times(int(maxRetries) + 1)

	s.logger.EXPECT().Warn("deadlock detected, will retry: %v", errDeadlock.Message).Times(6)

	err := s.retryingRepo.Update(s.ctx, s.model)

	s.Error(err)
	s.Equal(6, attempts)
	s.repo.AssertExpectations(s.T())
}

func (s *retryingRepositoryTestSuite) TestDelete_NonRetryable() {
	errPermanent := errors.New("non-retryable error")
	s.repo.EXPECT().Delete(mock.AnythingOfType("context.backgroundCtx"), s.model).Return(errPermanent).Once()
	s.logger.EXPECT().Warn("non-retryable DB error: %v", errPermanent)

	err := s.retryingRepo.Delete(s.ctx, s.model)

	s.EqualError(err, "non-retryable error")
	s.repo.AssertExpectations(s.T())
}

func TestRetryingRepositoryTestSuite(t *testing.T) {
	suite.Run(t, new(retryingRepositoryTestSuite))
}

func (s *retryingRepositoryTestSuite) TestDelete_WithRetry() {
	attempts := 0
	errDeadlock := &mysql.MySQLError{Number: dbRepo.DeadlockErrorCode, Message: "Deadlock on delete"}
	s.repo.On("Delete", mock.Anything, s.model).Return(func(ctx context.Context, m dbRepo.ModelBased) error {
		attempts++
		if attempts < 2 {
			return errDeadlock
		}

		return nil
	}).Times(2)
	s.logger.EXPECT().Warn("deadlock detected, will retry: %v", errDeadlock.Message)

	err := s.retryingRepo.Delete(s.ctx, s.model)

	s.NoError(err)
	s.repo.AssertExpectations(s.T())
	s.Equal(2, attempts)
}
