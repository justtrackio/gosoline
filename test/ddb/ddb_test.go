// +build integration

package ddb_test

import (
	"fmt"
	cfgMocks "github.com/applike/gosoline/pkg/cfg/mocks"
	"github.com/applike/gosoline/pkg/cloud"
	"github.com/applike/gosoline/pkg/ddb"
	"github.com/applike/gosoline/pkg/mdl"
	monMocks "github.com/applike/gosoline/pkg/mon/mocks"
	pkgTest "github.com/applike/gosoline/pkg/test"
	"github.com/applike/gosoline/pkg/tracing"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"os"
	"testing"
)

type TestData struct {
	Id   string `json:"id" ddb:"key=hash"`
	Data string `json:"data"`
}

type DdbTestSuite struct {
	suite.Suite
	mocks     *pkgTest.Mocks
	ddbConfig ddb.Settings
	repo      ddb.Repository
}

func (s *DdbTestSuite) SetupSuite() {
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

func (s *DdbTestSuite) TearDownSuite() {
	if s.mocks != nil {
		s.mocks.Shutdown()
		s.mocks = nil
	}
}

func (s *DdbTestSuite) SetupTest() {
	ddbEndpoint := fmt.Sprintf("http://%s:%d", s.mocks.ProvideDynamoDbHost("dynamodb"), s.mocks.ProvideDynamoDbPort("dynamodb"))

	config := new(cfgMocks.Config)
	config.On("GetBool", "aws_dynamoDb_autoCreate").Return(true)
	config.On("GetInt", "aws_sdk_retries").Return(3)
	config.On("UnmarshalKey", "tracing", &tracing.TracerSettings{})
	config.On("GetString", "aws_dynamoDb_endpoint").Return(ddbEndpoint)
	config.On("UnmarshalKey", "ddb.backoff", &cloud.BackoffSettings{}).Run(func(args mock.Arguments) {
		backoffSettings := args.Get(1).(*cloud.BackoffSettings)
		*backoffSettings = cloud.BackoffSettings{
			Enabled:  true,
			Blocking: true,
		}
	})
	config.On("GetString", "app_project").Return("gosoline")
	config.On("GetString", "env").Return("test")
	config.On("GetString", "app_family").Return("test")
	config.On("GetString", "app_name").Return("ddb-lock-test")

	logger := monMocks.NewLoggerMockedAll()

	s.ddbConfig = ddb.Settings{
		ModelId: mdl.ModelId{
			Name: "test-data",
		},
		Main: ddb.MainSettings{
			Model:              &TestData{},
			ReadCapacityUnits:  1,
			WriteCapacityUnits: 1,
		},
	}
	s.repo = ddb.NewRepository(config, logger, &s.ddbConfig)
}

func (s *DdbTestSuite) TestSetValidBackoffConfig() {
	s.True(s.ddbConfig.Backoff.Blocking)
	s.True(s.ddbConfig.Backoff.Enabled)
}

func TestDdb(t *testing.T) {
	suite.Run(t, new(DdbTestSuite))
}
