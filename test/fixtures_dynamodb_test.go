//+build integration

package test_test

import (
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/ddb"
	"github.com/applike/gosoline/pkg/fixtures"
	"github.com/applike/gosoline/pkg/mdl"
	"github.com/applike/gosoline/pkg/mon"
	"github.com/applike/gosoline/pkg/test"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"testing"
)

type FixturesDynamoDbSuite struct {
	suite.Suite
	logger mon.Logger
	db     *dynamodb.DynamoDB
	mocks  *test.Mocks
}

func (s *FixturesDynamoDbSuite) SetupSuite() {
	setup(s.T())
	mocks, err := test.Boot("test_configs/config.dynamodb.test.yml")

	if err != nil {
		assert.Fail(s.T(), "failed to boot mocks")

		return
	}

	s.mocks = mocks
	s.db = s.mocks.ProvideDynamoDbClient("dynamodb")
	s.logger = mon.NewLogger()
}

func (s *FixturesDynamoDbSuite) TearDownSuite() {
	s.mocks.Shutdown()
}

func TestFixturesDynamoDbSuite(t *testing.T) {
	suite.Run(t, new(FixturesDynamoDbSuite))
}

func (s FixturesDynamoDbSuite) TestDynamoDb() {
	config := cfg.New()
	err := config.Option(
		cfg.WithConfigFile("test_configs/config.dynamodb.test.yml", "yml"),
		cfg.WithConfigFile("test_configs/config.fixtures_dynamodb.test.yml", "yml"),
		cfg.WithConfigMap(map[string]interface{}{
			"aws_dynamoDb_endpoint": fmt.Sprintf("%s:%d", "http://172.17.0.1", s.mocks.Ports("dynamodb")["dynamodb"]),
		}),
	)

	assert.NoError(s.T(), err)

	loader := fixtures.NewFixtureLoader(config, s.logger)

	err = loader.Load(dynamoDbDisabledPurgeFixtures())
	assert.NoError(s.T(), err)

	gio, err := s.db.GetItem(&dynamodb.GetItemInput{
		Key: map[string]*dynamodb.AttributeValue{
			"Name": {
				S: aws.String("Ash"),
			},
		},
		TableName: aws.String("gosoline-test-integration-test-test-application-testModel"),
	})

	// should have created the item
	assert.NoError(s.T(), err)
	assert.Len(s.T(), gio.Item, 2, "2 attributes expected")
	assert.Equal(s.T(), "Ash", *(gio.Item["Name"].S))
	assert.Equal(s.T(), "10", *(gio.Item["Age"].N))

	qo, err := s.db.Query(&dynamodb.QueryInput{
		TableName:              aws.String("gosoline-test-integration-test-test-application-testModel"),
		IndexName:              aws.String("IDX_Age"),
		KeyConditionExpression: aws.String("Age = :v_age"),
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":v_age": {
				N: aws.String("10"),
			},
		},
	})

	// should have created global index
	assert.NoError(s.T(), err)
	assert.Len(s.T(), qo.Items, 1, "1 item expected")
}

func (s FixturesDynamoDbSuite) TestDynamoDbWithPurge() {
	config := cfg.New()
	err := config.Option(
		cfg.WithConfigFile("test_configs/config.dynamodb.test.yml", "yml"),
		cfg.WithConfigFile("test_configs/config.fixtures_dynamodb.test.yml", "yml"),
		cfg.WithConfigMap(map[string]interface{}{
			"aws_dynamoDb_endpoint": fmt.Sprintf("%s:%d", "http://172.17.0.1", s.mocks.Ports("dynamodb")["dynamodb"]),
		}),
	)

	assert.NoError(s.T(), err)

	loader := fixtures.NewFixtureLoader(config, s.logger)

	err = loader.Load(dynamoDbDisabledPurgeFixtures())
	assert.NoError(s.T(), err)

	gio, err := s.db.GetItem(&dynamodb.GetItemInput{
		Key: map[string]*dynamodb.AttributeValue{
			"Name": {
				S: aws.String("Ash"),
			},
		},
		TableName: aws.String("gosoline-test-integration-test-test-application-testModel"),
	})

	// should have created the item
	assert.NoError(s.T(), err)
	assert.Len(s.T(), gio.Item, 2, "2 attributes expected")
	assert.Equal(s.T(), "Ash", *(gio.Item["Name"].S))
	assert.Equal(s.T(), "10", *(gio.Item["Age"].N))

	err = loader.Load(dynamoDbEnabledPurgeFixtures())
	assert.NoError(s.T(), err)

	gio, err = s.db.GetItem(&dynamodb.GetItemInput{
		Key: map[string]*dynamodb.AttributeValue{
			"Name": {
				S: aws.String("Bash"),
			},
		},
		TableName: aws.String("gosoline-test-integration-test-test-application-testModel"),
	})

	// should have created the item
	assert.NoError(s.T(), err)
	assert.Len(s.T(), gio.Item, 2, "2 attributes expected")
	assert.Equal(s.T(), "Bash", *(gio.Item["Name"].S))
	assert.Equal(s.T(), "10", *(gio.Item["Age"].N))

	gio, err = s.db.GetItem(&dynamodb.GetItemInput{
		Key: map[string]*dynamodb.AttributeValue{
			"Name": {
				S: aws.String("Ash"),
			},
		},
		TableName: aws.String("gosoline-test-integration-test-test-application-testModel"),
	})

	// should have created the item
	assert.NoError(s.T(), err)
	assert.Nil(s.T(), gio.Item)

	qo, err := s.db.Query(&dynamodb.QueryInput{
		TableName:              aws.String("gosoline-test-integration-test-test-application-testModel"),
		IndexName:              aws.String("IDX_Age"),
		KeyConditionExpression: aws.String("Age = :v_age"),
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":v_age": {
				N: aws.String("10"),
			},
		},
	})

	// should have created global index
	assert.NoError(s.T(), err)
	assert.Len(s.T(), qo.Items, 1, "1 item expected")
}

func (s FixturesDynamoDbSuite) TestDynamoDbKvStore() {
	config := cfg.New()
	err := config.Option(
		cfg.WithConfigFile("test_configs/config.dynamodb.test.yml", "yml"),
		cfg.WithConfigFile("test_configs/config.fixtures_dynamodb.test.yml", "yml"),
		cfg.WithConfigMap(map[string]interface{}{
			"aws_dynamoDb_endpoint": fmt.Sprintf("%s:%d", "http://172.17.0.1", s.mocks.Ports("dynamodb")["dynamodb"]),
		}),
	)

	assert.NoError(s.T(), err)

	loader := fixtures.NewFixtureLoader(config, s.logger)

	err = loader.Load(dynamoDbKvStoreDisabledPurgeFixtures())
	assert.NoError(s.T(), err)

	gio, err := s.db.GetItem(&dynamodb.GetItemInput{
		Key: map[string]*dynamodb.AttributeValue{
			"key": {
				S: aws.String("Ash"),
			},
		},
		TableName: aws.String("gosoline-test-integration-test-test-application-kvstore-testModel"),
	})

	// should have created the item
	assert.NoError(s.T(), err)
	assert.Len(s.T(), gio.Item, 2, "2 attributes expected")
	assert.JSONEq(s.T(), `{"Name":"Ash","Age":10}`, *(gio.Item["value"].S))
}

func (s FixturesDynamoDbSuite) TestDynamoDbKvStoreWithPurge() {
	config := cfg.New()
	err := config.Option(
		cfg.WithConfigFile("test_configs/config.dynamodb.test.yml", "yml"),
		cfg.WithConfigFile("test_configs/config.fixtures_dynamodb.test.yml", "yml"),
		cfg.WithConfigMap(map[string]interface{}{
			"aws_dynamoDb_endpoint": fmt.Sprintf("%s:%d", "http://172.17.0.1", s.mocks.Ports("dynamodb")["dynamodb"]),
		}),
	)

	assert.NoError(s.T(), err)

	loader := fixtures.NewFixtureLoader(config, s.logger)

	err = loader.Load(dynamoDbKvStoreDisabledPurgeFixtures())
	assert.NoError(s.T(), err)

	gio, err := s.db.GetItem(&dynamodb.GetItemInput{
		Key: map[string]*dynamodb.AttributeValue{
			"key": {
				S: aws.String("Ash"),
			},
		},
		TableName: aws.String("gosoline-test-integration-test-test-application-kvstore-testModel"),
	})

	// should have created the item
	assert.NoError(s.T(), err)
	assert.Len(s.T(), gio.Item, 2, "2 attributes expected")
	assert.JSONEq(s.T(), `{"Name":"Ash","Age":10}`, *(gio.Item["value"].S))

	err = loader.Load(dynamoDbKvStoreEnabledPurgeFixtures())
	assert.NoError(s.T(), err)

	gio, err = s.db.GetItem(&dynamodb.GetItemInput{
		Key: map[string]*dynamodb.AttributeValue{
			"key": {
				S: aws.String("Bash"),
			},
		},
		TableName: aws.String("gosoline-test-integration-test-test-application-kvstore-testModel"),
	})

	// should have created the item
	assert.NoError(s.T(), err)
	assert.Len(s.T(), gio.Item, 2, "2 attributes expected")
	assert.JSONEq(s.T(), `{"Name":"Bash","Age":10}`, *(gio.Item["value"].S))

	gio, err = s.db.GetItem(&dynamodb.GetItemInput{
		Key: map[string]*dynamodb.AttributeValue{
			"key": {
				S: aws.String("Ash"),
			},
		},
		TableName: aws.String("gosoline-test-integration-test-test-application-kvstore-testModel"),
	})

	assert.NoError(s.T(), err)
	assert.Nil(s.T(), gio.Item, "no item expected")
}

type DynamoDbTestModel struct {
	Name string `ddb:"key=hash"`
	Age  uint   `ddb:"global=hash"`
}

func dynamoDbKvStoreDisabledPurgeFixtures() []*fixtures.FixtureSet {
	return []*fixtures.FixtureSet{
		{
			Enabled: true,
			Writer: fixtures.DynamoDbKvStoreFixtureWriterFactory(&mdl.ModelId{
				Project:     "gosoline",
				Environment: "test",
				Family:      "integration-test",
				Application: "test-application",
				Name:        "testModel",
			}),
			Fixtures: []interface{}{
				&fixtures.KvStoreFixture{
					Key:   "Ash",
					Value: &DynamoDbTestModel{Name: "Ash", Age: 10},
				},
			},
		},
	}
}

func dynamoDbKvStoreEnabledPurgeFixtures() []*fixtures.FixtureSet {
	return []*fixtures.FixtureSet{
		{
			Enabled: true,
			Purge:   true,
			Writer: fixtures.DynamoDbKvStoreFixtureWriterFactory(&mdl.ModelId{
				Project:     "gosoline",
				Environment: "test",
				Family:      "integration-test",
				Application: "test-application",
				Name:        "testModel",
			}),
			Fixtures: []interface{}{
				&fixtures.KvStoreFixture{
					Key:   "Bash",
					Value: &DynamoDbTestModel{Name: "Bash", Age: 10},
				},
			},
		},
	}
}

func dynamoDbDisabledPurgeFixtures() []*fixtures.FixtureSet {
	return []*fixtures.FixtureSet{
		{
			Enabled: true,
			Writer: fixtures.DynamoDbFixtureWriterFactory(&ddb.Settings{
				ModelId: mdl.ModelId{
					Project:     "gosoline",
					Environment: "test",
					Family:      "integration-test",
					Application: "test-application",
					Name:        "testModel",
				},
				Main: ddb.MainSettings{
					Model: DynamoDbTestModel{},
				},
				Global: []ddb.GlobalSettings{
					{
						Name:               "IDX_Age",
						Model:              DynamoDbTestModel{},
						ReadCapacityUnits:  1,
						WriteCapacityUnits: 1,
					},
				},
			}),
			Fixtures: []interface{}{
				&DynamoDbTestModel{Name: "Ash", Age: 10},
			},
		},
	}
}

func dynamoDbEnabledPurgeFixtures() []*fixtures.FixtureSet {
	return []*fixtures.FixtureSet{
		{
			Enabled: true,
			Purge:   true,
			Writer: fixtures.DynamoDbFixtureWriterFactory(&ddb.Settings{
				ModelId: mdl.ModelId{
					Project:     "gosoline",
					Environment: "test",
					Family:      "integration-test",
					Application: "test-application",
					Name:        "testModel",
				},
				Main: ddb.MainSettings{
					Model: DynamoDbTestModel{},
				},
				Global: []ddb.GlobalSettings{
					{
						Name:               "IDX_Age",
						Model:              DynamoDbTestModel{},
						ReadCapacityUnits:  1,
						WriteCapacityUnits: 1,
					},
				},
			}),
			Fixtures: []interface{}{
				&DynamoDbTestModel{Name: "Bash", Age: 10},
			},
		},
	}
}
