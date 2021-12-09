package conc

import (
	"context"
	"fmt"
	"time"

	"github.com/cenkalti/backoff"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/clock"
	"github.com/justtrackio/gosoline/pkg/cloud/aws/dynamodb"
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
	backOff         backoff.BackOff
	clock           clock.Clock
	uuidSource      uuid.Uuid
	defaultLockTime time.Duration
	domain          string
}

func NewDdbLockProvider(ctx context.Context, config cfg.Config, logger log.Logger, settings DistributedLockSettings) (DistributedLockProvider, error) {
	ddbSettings := &ddb.Settings{
		ModelId: mdl.ModelId{
			Name: "locks",
		},
		Main: ddb.MainSettings{
			Model:              &DdbLockItem{},
			ReadCapacityUnits:  1,
			WriteCapacityUnits: 1,
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

	return NewDdbLockProviderWithInterfaces(
		logger,
		repo,
		backoff.NewExponentialBackOff(),
		clock.NewRealClock(),
		uuid.New(),
		settings,
	), nil
}

func NewDdbLockProviderWithInterfaces(
	logger log.Logger,
	repo ddb.Repository,
	backOff backoff.BackOff,
	clock clock.Clock,
	uuidSource uuid.Uuid,
	settings DistributedLockSettings,
) DistributedLockProvider {
	return &ddbLockProvider{
		logger:          logger.WithChannel("ddbLock"),
		repo:            repo,
		backOff:         backOff,
		clock:           clock,
		uuidSource:      uuidSource,
		defaultLockTime: settings.DefaultLockTime,
		domain:          settings.Domain,
	}
}

func (m *ddbLockProvider) Acquire(ctx context.Context, resource string) (DistributedLock, error) {
	resource = fmt.Sprintf("%s-%s", m.domain, resource)
	token := m.uuidSource.NewV4()

	var lock *ddbLock
	err := backoff.Retry(func() error {
		now := m.clock.Now()
		// ddb does return expired items if they have not yet been deleted
		// to account for potential clock skew, we treat items that have been expired by at least a minute as deleted
		ttlThreshold := now.Unix() - 60
		expires := now.Add(m.defaultLockTime).Unix()
		qb := m.repo.PutItemBuilder().
			WithCondition(ddb.AttributeNotExists("resource").Or(ddb.Lt("ttl", ttlThreshold)))

		result, err := m.repo.PutItem(ctx, qb, &DdbLockItem{
			Resource: resource,
			Token:    token,
			Ttl:      expires,
		})

		if exec.IsRequestCanceled(err) {
			return backoff.Permanent(err)
		}

		if err != nil {
			return err
		}

		if result.ConditionalCheckFailed {
			return ErrOwnedLock
		}

		m.logger.WithContext(ctx).WithFields(log.Fields{
			"ddb_lock_token":    token,
			"ddb_lock_resource": resource,
		}).Debug("acquired lock")

		lock = newDdbLock(m, ctx, resource, token, expires)
		lock.forkWatcher()

		return nil
	}, m.backOff)

	return lock, err
}

func (m *ddbLockProvider) renew(ctx context.Context, lockTime time.Duration, resource string, token string) error {
	return backoff.Retry(func() error {
		qb := m.repo.UpdateItemBuilder().
			WithHash(resource).
			WithCondition(ddb.AttributeExists("resource").And(ddb.Eq("token", token)))

		result, err := m.repo.UpdateItem(ctx, qb, &DdbLockItem{
			Resource: resource,
			Token:    token,
			Ttl:      m.clock.Now().Add(lockTime).Unix(),
		})

		if exec.IsRequestCanceled(err) {
			return backoff.Permanent(err)
		}

		if err != nil {
			return err
		}

		if result.ConditionalCheckFailed {
			return backoff.Permanent(ErrNotOwned)
		}

		m.logger.WithContext(ctx).WithFields(log.Fields{
			"ddb_lock_token":    token,
			"ddb_lock_resource": resource,
		}).Debug("renewed lock")

		return nil
	}, m.backOff)
}

func (m *ddbLockProvider) release(ctx context.Context, resource string, token string) error {
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
		return ErrNotOwned
	}

	m.logger.WithContext(ctx).WithFields(log.Fields{
		"ddb_lock_token":    token,
		"ddb_lock_resource": resource,
	}).Debug("released lock")

	return nil
}
