package ddb

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/clock"
	"github.com/justtrackio/gosoline/pkg/cloud/aws/dynamodb"
	"github.com/justtrackio/gosoline/pkg/conc"
	"github.com/justtrackio/gosoline/pkg/ddb"
	"github.com/justtrackio/gosoline/pkg/exec"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/mdl"
	"github.com/justtrackio/gosoline/pkg/uuid"
)

type DdbLockItem struct {
	// unique name of the locked resource
	Resource string `json:"resource" ddb:"key=hash"`
	// token to ensure we are releasing a lock we own
	Token string `json:"token"`
	// ttl until the lock should be released automatically
	Ttl int64 `json:"ttl" ddb:"ttl=enabled"`
}

type ddbLockProvider struct {
	logger          log.Logger
	repo            ddb.Repository
	executor        exec.Executor
	clock           clock.Clock
	uuidSource      uuid.Uuid
	defaultLockTime time.Duration
	domain          string
}

func NewDdbLockProvider(
	ctx context.Context,
	config cfg.Config,
	logger log.Logger,
	settings conc.DistributedLockSettings,
) (conc.DistributedLockProvider, error) {
	ddbSettings := &ddb.Settings{
		ModelId: mdl.ModelId{
			Project:     settings.Project,
			Environment: settings.Environment,
			Family:      settings.Family,
			Group:       settings.Group,
			Application: settings.Application,
			Name:        "locks",
		},
		Main: ddb.MainSettings{
			Model: &DdbLockItem{},
		},
	}

	ddbClientOption := func(cfg *dynamodb.ClientConfig) {
		cfg.Settings.Backoff.CancelDelay = 0
	}

	var err error
	var repo ddb.Repository

	if repo, err = ddb.NewRepository(ctx, config, logger, ddbSettings, ddbClientOption); err != nil {
		return nil, fmt.Errorf("can not create ddb repository: %w", err)
	}

	res := &exec.ExecutableResource{
		Type: "ddbLock",
		Name: settings.Domain,
	}
	executor := exec.NewBackoffExecutor(logger, res, &settings.Backoff, []exec.ErrorChecker{CheckDdbLockError})

	return NewDdbLockProviderWithInterfaces(
		logger,
		repo,
		executor,
		clock.Provider,
		uuid.New(),
		settings,
	), nil
}

func NewDdbLockProviderWithInterfaces(
	logger log.Logger,
	repo ddb.Repository,
	executor exec.Executor,
	clock clock.Clock,
	uuidSource uuid.Uuid,
	settings conc.DistributedLockSettings,
) conc.DistributedLockProvider {
	return &ddbLockProvider{
		logger:          logger.WithChannel("ddbLock"),
		repo:            repo,
		executor:        executor,
		clock:           clock,
		uuidSource:      uuidSource,
		defaultLockTime: settings.DefaultLockTime,
		domain:          settings.Domain,
	}
}

func (m *ddbLockProvider) Acquire(ctx context.Context, resource string) (conc.DistributedLock, error) {
	resource = fmt.Sprintf("%s-%s", m.domain, resource)
	token := m.uuidSource.NewV4()

	var lock *ddbLock
	_, err := m.executor.Execute(ctx, func(ctx context.Context) (any, error) {
		now := m.clock.Now()
		// ddb does return expired items if they have not yet been deleted
		// to account for potential clock skew, we treat items that have been expired by at least five seconds as deleted
		ttlThreshold := now.Unix() - 5
		expires := now.Add(m.defaultLockTime)
		qb := m.repo.PutItemBuilder().
			WithCondition(ddb.AttributeNotExists("resource").Or(ddb.Lt("ttl", ttlThreshold)))

		result, err := m.repo.PutItem(ctx, qb, &DdbLockItem{
			Resource: resource,
			Token:    token,
			Ttl:      expires.Unix(),
		})

		if err != nil {
			return nil, err
		}

		if result.ConditionalCheckFailed {
			return nil, conc.ErrLockOwned
		}

		m.logger.WithFields(log.Fields{
			"ddb_lock_token":    token,
			"ddb_lock_resource": resource,
		}).Debug(ctx, "acquired lock")

		lock = NewDdbLockFromInterfaces(m, m.clock, m.logger, ctx, resource, token, expires)
		go lock.runWatcher()

		return nil, nil
	})

	return lock, err
}

func (m *ddbLockProvider) RenewLock(ctx context.Context, lockTime time.Duration, resource string, token string) (expiry time.Time, err error) {
	_, err = m.executor.Execute(ctx, func(ctx context.Context) (any, error) {
		qb := m.repo.UpdateItemBuilder().
			WithHash(resource).
			WithCondition(ddb.AttributeExists("resource").And(ddb.Eq("token", token)))

		expiry = m.clock.Now().Add(lockTime)
		result, err := m.repo.UpdateItem(ctx, qb, &DdbLockItem{
			Resource: resource,
			Token:    token,
			Ttl:      expiry.Unix(),
		})

		if err != nil {
			return nil, fmt.Errorf("failed to renew lock: %w", err)
		}

		if result.ConditionalCheckFailed {
			return nil, conc.ErrLockNotOwned
		}

		m.logger.WithFields(log.Fields{
			"ddb_lock_token":    token,
			"ddb_lock_resource": resource,
		}).Debug(ctx, "renewed lock")

		return nil, nil
	})

	return expiry, err
}

func (m *ddbLockProvider) ReleaseLock(ctx context.Context, resource string, token string) error {
	qb := m.repo.DeleteItemBuilder().
		WithHash(resource).
		WithCondition(ddb.AttributeExists("resource").And(ddb.Eq("token", token)))

	result, err := m.repo.DeleteItem(ctx, qb, &DdbLockItem{
		Resource: resource,
		Token:    token,
	})
	if err != nil {
		return err
	}

	if result.ConditionalCheckFailed {
		return conc.ErrLockNotOwned
	}

	m.logger.WithFields(log.Fields{
		"ddb_lock_token":    token,
		"ddb_lock_resource": resource,
	}).Debug(ctx, "released lock")

	return nil
}

func CheckDdbLockError(_ any, err error) exec.ErrorType {
	if exec.IsRequestCanceled(err) || errors.Is(err, conc.ErrLockNotOwned) {
		return exec.ErrorTypePermanent
	}

	return exec.ErrorTypeRetryable
}
