//go:build integration
// +build integration

package conc_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/justtrackio/gosoline/pkg/clock"
	"github.com/justtrackio/gosoline/pkg/conc"
	"github.com/justtrackio/gosoline/pkg/conc/ddb"
	"github.com/justtrackio/gosoline/pkg/exec"
	"github.com/justtrackio/gosoline/pkg/test/suite"
)

type DdbLockTestSuite struct {
	suite.Suite
	provider conc.DistributedLockProvider
}

func (s *DdbLockTestSuite) SetupSuite() []suite.Option {
	return []suite.Option{
		suite.WithClockProvider(clock.NewRealClock()),
		suite.WithLogLevel("debug"),
		suite.WithConfigFile("./config.dist.yml"),
	}
}

func (s *DdbLockTestSuite) SetupTest() (err error) {
	s.provider, err = ddb.NewDdbLockProvider(s.Env().Context(), s.Env().Config(), s.Env().Logger(), conc.DistributedLockSettings{
		Backoff: exec.BackoffSettings{
			MaxAttempts:    0,
			MaxElapsedTime: 0,
		},
		DefaultLockTime: time.Minute * 10,
		Domain:          fmt.Sprintf("test%d", time.Now().Unix()),
	})

	return
}

func (s *DdbLockTestSuite) TestLockAndRelease() {
	// Case 1: Acquire a lock and release it again
	ctx, cancel := context.WithTimeout(s.T().Context(), time.Minute)
	defer cancel()

	l, err := s.provider.Acquire(ctx, "a")
	s.NoError(err)
	err = l.Release()
	s.NoError(err)
}

func (s *DdbLockTestSuite) TestAcquireTwiceFails() {
	// Case 2: Acquire a lock, then try to acquire it again. Second call fails
	ctx, cancel := context.WithTimeout(s.T().Context(), time.Minute)
	defer cancel()

	l, err := s.provider.Acquire(ctx, "a")
	s.NoError(err)

	ctx2, cancel2 := context.WithTimeout(s.T().Context(), time.Second)
	defer cancel2()

	_, err = s.provider.Acquire(ctx2, "a")
	s.Error(err)
	s.True(exec.IsRequestCanceled(err))
	err = l.Release()
	s.NoError(err)
}

func (s *DdbLockTestSuite) TestAcquireRenewWorks() {
	// Case 3: Acquire a lock, then renew it, try to lock it again (should fail), release it
	ctx, cancel := context.WithTimeout(s.T().Context(), time.Minute)
	defer cancel()

	l, err := s.provider.Acquire(ctx, "a")
	s.NoError(err)
	err = l.Renew(ctx, time.Hour)
	s.NoError(err)

	ctx2, cancel2 := context.WithTimeout(s.T().Context(), time.Second)
	defer cancel2()

	_, err = s.provider.Acquire(ctx2, "a")
	s.Error(err)
	s.True(exec.IsRequestCanceled(err))
	err = l.Release()
	s.NoError(err)
}

func (s *DdbLockTestSuite) TestReleaseTwiceFails() {
	// Case 4: try to release a lock twice
	ctx, cancel := context.WithTimeout(s.T().Context(), time.Minute)
	defer cancel()

	l, err := s.provider.Acquire(ctx, "a")
	s.NoError(err)
	err = l.Release()
	s.NoError(err)
	err = l.Release()
	s.Error(err)
	s.Equal(conc.ErrNotOwned, err)
}

func (s *DdbLockTestSuite) TestRenewAfterReleaseFails() {
	// Case 5: try to renew a lock after releasing it
	ctx, cancel := context.WithTimeout(s.T().Context(), time.Minute)
	defer cancel()

	l, err := s.provider.Acquire(ctx, "a")
	s.NoError(err)
	err = l.Release()
	s.NoError(err)
	err = l.Renew(ctx, time.Hour)
	s.Error(err)
	s.Equal(conc.ErrNotOwned, err)
}

func (s *DdbLockTestSuite) TestAcquireDifferentResources() {
	// Case 6: try to acquire two different resources
	ctx, cancel := context.WithTimeout(s.T().Context(), time.Minute)
	defer cancel()

	l1, err := s.provider.Acquire(ctx, "a")
	s.NoError(err)
	l2, err := s.provider.Acquire(ctx, "b")
	s.NoError(err)
	err = l1.Release()
	s.NoError(err)
	err = l2.Release()
	s.NoError(err)
}

func TestDdbLockManager(t *testing.T) {
	suite.Run(t, new(DdbLockTestSuite))
}
