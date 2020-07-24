package conc

import (
	"context"
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/ddb"
	"github.com/applike/gosoline/pkg/exec"
	"github.com/applike/gosoline/pkg/mdl"
	"github.com/applike/gosoline/pkg/mon"
	"github.com/applike/gosoline/pkg/uuid"
	"github.com/cenkalti/backoff"
	"github.com/jonboulle/clockwork"
	"time"
)

type DdbLockItem struct {
	// unique name of the locked resource
	Resource string `json:"resource" ddb:"key=hash"`
	// token to ensure we are releasing a lock we are owning
	Token string `json:"token"`
	// ttl until the lock should be released automatically
	Ttl int64 `json:"ttl" ddb:"ttl=enabled"`
}

type ddbLockProvider struct {
	logger          mon.Logger
	repo            ddb.Repository
	backOff         backoff.BackOff
	clock           clockwork.Clock
	uuidSource      uuid.Uuid
	defaultLockTime time.Duration
	domain          string
}

func NewDdbLockProvider(config cfg.Config, logger mon.Logger, settings DistributedLockSettings) DistributedLockProvider {
	repo := ddb.NewRepository(config, logger, &ddb.Settings{
		ModelId: mdl.ModelId{
			Name: "locks",
		},
		Backoff: settings.Backoff,
		Main: ddb.MainSettings{
			Model:              &DdbLockItem{},
			ReadCapacityUnits:  1,
			WriteCapacityUnits: 1,
		},
	})

	backoffConfig := backoff.NewExponentialBackOff()
	if settings.Backoff.InitialInterval > 0 {
		backoffConfig.InitialInterval = settings.Backoff.InitialInterval
	}

	if settings.Backoff.RandomizationFactor > 0 {
		backoffConfig.RandomizationFactor = settings.Backoff.RandomizationFactor
	}

	if settings.Backoff.Multiplier > 0 {
		backoffConfig.Multiplier = settings.Backoff.Multiplier
	}

	if settings.Backoff.MaxInterval > 0 {
		backoffConfig.MaxInterval = settings.Backoff.MaxInterval
	}

	if settings.Backoff.MaxElapsedTime > 0 {
		backoffConfig.MaxElapsedTime = settings.Backoff.MaxElapsedTime
	}

	if settings.Backoff.Blocking {
		backoffConfig.MaxElapsedTime = 0
	}

	return NewDdbLockProviderWithInterfaces(
		logger,
		repo,
		backoffConfig,
		clockwork.NewRealClock(),
		uuid.New(),
		settings,
	)
}

func NewDdbLockProviderWithInterfaces(
	logger mon.Logger,
	repo ddb.Repository,
	backOff backoff.BackOff,
	clock clockwork.Clock,
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
		// to account for potential clock skew, we treat items which have been
		// expired by at least a minute as deleted
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
			return OwnedLockError
		}

		m.logger.WithContext(ctx).WithFields(mon.Fields{
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
			return backoff.Permanent(NotOwnedError)
		}

		m.logger.WithContext(ctx).WithFields(mon.Fields{
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
		return NotOwnedError
	}

	m.logger.WithContext(ctx).WithFields(mon.Fields{
		"ddb_lock_token":    token,
		"ddb_lock_resource": resource,
	}).Debug("released lock")

	return nil
}
