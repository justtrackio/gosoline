//go:build integration

package conc_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/clock"
	"github.com/justtrackio/gosoline/pkg/conc"
	"github.com/justtrackio/gosoline/pkg/conc/ddb"
	ddbRepo "github.com/justtrackio/gosoline/pkg/ddb"
	"github.com/justtrackio/gosoline/pkg/exec"
	"github.com/justtrackio/gosoline/pkg/mdl"
	"github.com/justtrackio/gosoline/pkg/test/suite"
)

type DdbLockTestSuite struct {
	suite.Suite
	clock    clock.FakeClock
	provider conc.DistributedLockProvider
	repo     ddbRepo.Repository
	domain   string
}

func (s *DdbLockTestSuite) SetupSuite() []suite.Option {
	s.clock = clock.NewFakeClockAt(time.Now().UTC())

	return []suite.Option{
		suite.WithClockProvider(s.clock),
		suite.WithLogLevel("debug"),
		suite.WithConfigFile("./config.dist.yml"),
		suite.WithSharedEnvironment(),
	}
}

func (s *DdbLockTestSuite) SetupTest() error {
	identity, err := cfg.GetAppIdentity(s.Env().Config())
	if err != nil {
		return fmt.Errorf("failed to get app identity: %w", err)
	}

	s.domain = fmt.Sprintf("test%d", s.clock.Now().UnixNano())
	s.provider, err = ddb.NewDdbLockProvider(s.Env().Context(), s.Env().Config(), s.Env().Logger(), conc.DistributedLockSettings{
		Identity: identity,
		Backoff: exec.BackoffSettings{
			CancelDelay:     0,
			InitialInterval: time.Millisecond * 25,
			MaxAttempts:     2,
			MaxElapsedTime:  0,
			MaxInterval:     time.Millisecond * 100,
		},
		DefaultLockTime: time.Minute * 10,
		Domain:          s.domain,
	})
	if err != nil {
		return err
	}

	s.repo, err = ddbRepo.NewRepository(s.Env().Context(), s.Env().Config(), s.Env().Logger(), &ddbRepo.Settings{
		ModelId: mdl.ModelId{
			Name:        "locks",
			Env:         identity.Env,
			Application: identity.Name,
			Tags:        identity.Tags,
		},
		Main: ddbRepo.MainSettings{
			Model: &ddb.DdbLockItem{},
		},
	})

	return err
}

func (s *DdbLockTestSuite) TestLockAndRelease() {
	// Case 1: Acquire a lock and release it again
	l, err := s.provider.Acquire(s.T().Context(), s.T().Name())
	s.NoError(err)
	err = l.Release()
	s.NoError(err)
}

func (s *DdbLockTestSuite) TestAcquireTwiceFails() {
	// Case 2: Acquire a lock, then try to acquire it again. Second call fails
	l, err := s.provider.Acquire(s.T().Context(), s.T().Name())
	s.NoError(err)

	_, err = s.provider.Acquire(s.T().Context(), s.T().Name())
	s.Error(err)
	s.True(errors.Is(err, conc.ErrLockOwned), "Error is: %v", err)
	err = l.Release()
	s.NoError(err)
}

func (s *DdbLockTestSuite) TestAcquireRenewWorks() {
	// Case 3: Acquire a lock, renew it, verify the stored ttl changed and the renewed lease remains effective.
	ctx := s.T().Context()
	resource := s.T().Name()
	fullResource := fmt.Sprintf("%s-%s", s.domain, resource)
	renewLockTime := time.Hour

	l, err := s.provider.Acquire(ctx, resource)
	s.NoError(err)

	itemBeforeRenew := s.getLockItem(ctx, fullResource)
	originalExpiry := time.Unix(itemBeforeRenew.Ttl, 0)

	s.clock.Advance(time.Minute)
	err = l.Renew(ctx, renewLockTime)
	s.NoError(err)

	itemAfterRenew := s.getLockItem(ctx, fullResource)
	s.Greater(itemAfterRenew.Ttl, itemBeforeRenew.Ttl)
	s.Equal(s.clock.Now().Add(renewLockTime).Unix(), itemAfterRenew.Ttl)

	s.clock.Advance(originalExpiry.Add(6 * time.Second).Sub(s.clock.Now()))
	_, err = s.provider.Acquire(ctx, resource)
	s.Error(err)
	s.True(errors.Is(err, conc.ErrLockOwned), "Error is: %v", err)

	err = l.Release()
	s.NoError(err)

	l, err = s.provider.Acquire(ctx, resource)
	s.NoError(err)
	s.NoError(l.Release())
}

func (s *DdbLockTestSuite) TestReleaseTwiceFails() {
	// Case 4: try to release a lock twice
	l, err := s.provider.Acquire(s.T().Context(), s.T().Name())
	s.NoError(err)
	err = l.Release()
	s.NoError(err)
	err = l.Release()
	s.Error(err)
	s.Equal(conc.ErrLockNotOwned, err)
}

func (s *DdbLockTestSuite) TestRenewAfterReleaseFails() {
	// Case 5: try to renew a lock after releasing it
	l, err := s.provider.Acquire(s.T().Context(), s.T().Name())
	s.NoError(err)
	err = l.Release()
	s.NoError(err)
	err = l.Renew(s.T().Context(), time.Hour)
	s.Error(err)
	s.Equal(conc.ErrLockNotOwned, err)
}

func (s *DdbLockTestSuite) TestReleaseAfterAcquireContextCanceled() {
	// Case 6: releasing still works after the acquire context was canceled.
	ctx, cancel := context.WithCancel(s.T().Context())
	l, err := s.provider.Acquire(ctx, s.T().Name())
	s.NoError(err)

	cancel()
	err = l.Release()
	s.NoError(err)

	l, err = s.provider.Acquire(s.T().Context(), s.T().Name())
	s.NoError(err)
	s.NoError(l.Release())
}

func (s *DdbLockTestSuite) TestAcquireDifferentResources() {
	// Case 7: try to acquire two different resources
	l1, err := s.provider.Acquire(s.T().Context(), s.T().Name())
	s.NoError(err)
	l2, err := s.provider.Acquire(s.T().Context(), s.T().Name()+"-other")
	s.NoError(err)
	err = l1.Release()
	s.NoError(err)
	err = l2.Release()
	s.NoError(err)
}

func TestDdbLockManager(t *testing.T) {
	suite.Run(t, new(DdbLockTestSuite))
}

func (s *DdbLockTestSuite) getLockItem(ctx context.Context, resource string) *ddb.DdbLockItem {
	item := &ddb.DdbLockItem{}
	result, err := s.repo.GetItem(ctx, s.repo.GetItemBuilder().
		WithHash(resource).
		WithConsistentRead(true).
		DisableTtlFilter(), item)

	s.NoError(err)
	s.True(result.IsFound)

	return item
}
