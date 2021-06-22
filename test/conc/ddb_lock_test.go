// +build integration

package conc_test

import (
	"context"
	"fmt"
	cfgMocks "github.com/applike/gosoline/pkg/cfg/mocks"
	"github.com/applike/gosoline/pkg/conc"
	"github.com/applike/gosoline/pkg/exec"
	logMocks "github.com/applike/gosoline/pkg/log/mocks"
	pkgTest "github.com/applike/gosoline/pkg/test"
	"github.com/applike/gosoline/pkg/tracing"
	"github.com/stretchr/testify/suite"
	"os"
	"testing"
	"time"
)

type DdbLockTestSuite struct {
	suite.Suite
	mocks    *pkgTest.Mocks
	provider conc.DistributedLockProvider
}

func (s *DdbLockTestSuite) SetupSuite() {
	err := os.Setenv("AWS_ACCESS_KEY_ID", "a")
	s.NoError(err)
	err = os.Setenv("AWS_SECRET_ACCESS_KEY", "b")
	s.NoError(err)

	mocks, err := pkgTest.Boot("../test_configs/config.dynamodb.test.yml")

	if err != nil {
		if mocks != nil {
			mocks.Shutdown()
		}

		s.Fail("failed to boot mocks: %s", err.Error())

		return
	}

	s.mocks = mocks
}

func (s *DdbLockTestSuite) TearDownSuite() {
	if s.mocks != nil {
		s.mocks.Shutdown()
		s.mocks = nil
	}
}

func (s *DdbLockTestSuite) SetupTest() {
	ddbEndpoint := fmt.Sprintf("http://%s:%d", s.mocks.ProvideDynamoDbHost("dynamodb"), s.mocks.ProvideDynamoDbPort("dynamodb"))

	config := new(cfgMocks.Config)
	config.On("GetBool", "aws_dynamoDb_autoCreate").Return(true)
	config.On("GetInt", "aws_sdk_retries").Return(3)
	config.On("UnmarshalKey", "tracing", &tracing.TracerSettings{})
	config.On("GetString", "aws_dynamoDb_endpoint").Return(ddbEndpoint)
	config.On("UnmarshalKey", "ddb.backoff", &exec.BackoffSettings{})
	config.On("GetString", "app_project").Return("gosoline")
	config.On("GetString", "env").Return("test")
	config.On("GetString", "app_family").Return("test")
	config.On("GetString", "app_name").Return("ddb-lock-test")

	logger := logMocks.NewLoggerMockedAll()

	provider, err := conc.NewDdbLockProvider(config, logger, conc.DistributedLockSettings{
		Backoff: exec.BackoffSettings{
			Enabled:  true,
			Blocking: true,
		},
		DefaultLockTime: time.Second * 3,
		Domain:          fmt.Sprintf("test%d", time.Now().Unix()),
	})
	s.NoError(err)

	s.provider = provider
}

func (s *DdbLockTestSuite) TestLockAndRelease() {
	// Case 1: Acquire a lock and release it again
	ctx, _ := context.WithTimeout(context.Background(), time.Second*5)
	l, err := s.provider.Acquire(ctx, "a")
	s.NoError(err)
	err = l.Release()
	s.NoError(err)
}

func (s *DdbLockTestSuite) TestAcquireTwiceFails() {
	// Case 2: Acquire a lock, then try to acquire it again. Second call fails
	ctx, _ := context.WithTimeout(context.Background(), time.Second*5)
	l, err := s.provider.Acquire(ctx, "a")
	s.NoError(err)
	ctx2, _ := context.WithTimeout(context.Background(), time.Second)
	_, err = s.provider.Acquire(ctx2, "a")
	s.Error(err)
	s.True(exec.IsRequestCanceled(err))
	err = l.Release()
	s.NoError(err)
}

func (s *DdbLockTestSuite) TestAcquireRenewWorks() {
	// Case 3: Acquire a lock, then renew it, sleep some time, try to lock it again (should fail), release it
	ctx, _ := context.WithTimeout(context.Background(), time.Second*15)
	l, err := s.provider.Acquire(ctx, "a")
	s.NoError(err)
	time.Sleep(time.Second * 1)
	err = l.Renew(ctx, time.Second*10)
	s.NoError(err)
	time.Sleep(time.Second * 4)
	ctx2, _ := context.WithTimeout(context.Background(), time.Second)
	_, err = s.provider.Acquire(ctx2, "a")
	s.Error(err)
	s.True(exec.IsRequestCanceled(err))
	err = l.Release()
	s.NoError(err)
}

func (s *DdbLockTestSuite) TestReleaseTwiceFails() {
	// Case 4: try to release a lock twice
	ctx, _ := context.WithTimeout(context.Background(), time.Second*5)
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
	ctx, _ := context.WithTimeout(context.Background(), time.Second*5)
	l, err := s.provider.Acquire(ctx, "a")
	s.NoError(err)
	err = l.Release()
	s.NoError(err)
	err = l.Renew(ctx, time.Minute)
	s.Error(err)
	s.Equal(conc.ErrNotOwned, err)
}

func (s *DdbLockTestSuite) TestAcquireDifferentResources() {
	// Case 6: try to acquire two different resources
	ctx, _ := context.WithTimeout(context.Background(), time.Second*5)
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
