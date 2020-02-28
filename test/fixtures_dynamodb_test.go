//+build integration

package test_test

import (
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
	db     *dynamodb.DynamoDB
	logger mon.Logger
}

func (s *FixturesDynamoDbSuite) SetupSuite() {
	setup(s.T())
	test.Boot("test_configs/config.dynamodb.test.yml")
	s.db = test.ProvideDynamoDbClient("dynamodb")
	s.logger = mon.NewLogger()
}

func (s *FixturesDynamoDbSuite) TearDownSuite() {
	test.Shutdown()
}

func TestFixturesDynamoDbSuite(t *testing.T) {
	suite.Run(t, new(FixturesDynamoDbSuite))
}

func (s FixturesDynamoDbSuite) TestDynamoDb() {
	config := configFromFiles("test_configs/config.dynamodb.test.yml", "test_configs/config.fixtures_dynamodb.test.yml")

	loader := fixtures.NewFixtureLoader(config, s.logger)

	err := loader.Load(dynamoDbFixtures())
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

func (s FixturesDynamoDbSuite) TestDynamoDbKvStore() {
	config := configFromFiles("test_configs/config.dynamodb.test.yml", "test_configs/config.fixtures_dynamodb.test.yml")

	loader := fixtures.NewFixtureLoader(config, s.logger)

	err := loader.Load(dynamoDbKvStoreFixtures())
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

type DynamoDbTestModel struct {
	Name string `ddb:"key=hash"`
	Age  uint   `ddb:"global=hash"`
}

func dynamoDbKvStoreFixtures() []*fixtures.FixtureSet {
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

func dynamoDbFixtures() []*fixtures.FixtureSet {
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

func configFromFiles(filePaths ...string) cfg.Config {
	config := cfg.New()

	for _, filePath := range filePaths {
		err := cfg.WithConfigFile(filePath, "yml")(config)

		if err != nil {
			panic("could not find config file " + filePath)
		}
	}

	return config
}
