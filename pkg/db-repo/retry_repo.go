package db_repo

import (
	"context"
	"errors"

	"github.com/cenkalti/backoff/v4"
	"github.com/go-sql-driver/mysql"
	"github.com/justtrackio/gosoline/pkg/log"
)

const (
	DeadlockErrorCode = 1213
	DefaultRetries    = 5
)

type RetryEvalFunc func(logger log.Logger, err error) error

type RetryingRepository struct {
	Repository

	logger        log.Logger
	retryEvalFunc RetryEvalFunc
	backoffConfig backoff.BackOff
}

func NewRetryingRepository(logger log.Logger, base Repository, retryEvalFunc RetryEvalFunc, defaultBackoffConfig backoff.BackOff) *RetryingRepository {
	return &RetryingRepository{
		Repository:    base,
		logger:        logger,
		retryEvalFunc: retryEvalFunc,
		backoffConfig: defaultBackoffConfig,
	}
}

func (r *RetryingRepository) getRetryEvalFunc() RetryEvalFunc {
	if r.retryEvalFunc == nil {
		return DefaultRetryEvalFunc
	}

	return r.retryEvalFunc
}

func (r *RetryingRepository) Create(ctx context.Context, value ModelBased) error {
	if err := r.Repository.Create(ctx, value); err != nil {
		return err
	}

	return nil
}

func (r *RetryingRepository) Update(ctx context.Context, value ModelBased) error {
	cb := backoff.WithContext(r.backoffConfig, ctx)
	retryEvaluator := r.getRetryEvalFunc()

	return backoff.Retry(func() error {
		err := r.Repository.Update(ctx, value)
		if err != nil {
			return retryEvaluator(r.logger, err)
		}

		return nil
	}, cb)
}

func (r *RetryingRepository) Delete(ctx context.Context, value ModelBased) error {
	cb := backoff.WithContext(r.backoffConfig, ctx)
	retryEvaluator := r.getRetryEvalFunc()

	return backoff.Retry(func() error {
		err := r.Repository.Delete(ctx, value)
		if err != nil {
			return retryEvaluator(r.logger, err)
		}

		return nil
	}, cb)
}

func DefaultRetryEvalFunc(logger log.Logger, err error) error {
	var mysqlErr *mysql.MySQLError

	if errors.As(err, &mysqlErr) && mysqlErr.Number == DeadlockErrorCode {
		logger.Warn("deadlock detected, will retry: %v", mysqlErr.Message)

		return err
	}

	logger.Warn("non-retryable DB error: %v", err)

	return backoff.Permanent(err)
}

func DefaultBackoffConfig() backoff.BackOff {
	b := backoff.WithMaxRetries(backoff.NewExponentialBackOff(), DefaultRetries)

	return b
}
