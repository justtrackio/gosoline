//+build integration

package test_test

import (
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/fixtures"
	"github.com/applike/gosoline/pkg/mdl"
	"github.com/applike/gosoline/pkg/mon"
	"github.com/applike/gosoline/pkg/test"
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
	loader := fixtures.NewFixtureLoader(dynamoDbFixtures())

	config := configFromFiles("test_configs/config.dynamodb.test.yml", "test_configs/config.fixtures_dynamodb.test.yml")

	err := loader.Load(config, s.logger)
	assert.NoError(s.T(), err)

	gio, err := s.db.GetItem(&dynamodb.GetItemInput{
		Key: map[string]*dynamodb.AttributeValue{
			"Name": {
				S: mdl.String("Ash"),
			},
		},
		TableName: mdl.String("gosoline-test-integration-test-test-application-testModel"),
	})

	// should have created the item
	assert.NoError(s.T(), err)
	assert.Len(s.T(), gio.Item, 2, "2 attributes expected")
	assert.Equal(s.T(), "Ash", *(gio.Item["Name"].S))
	assert.Equal(s.T(), "10", *(gio.Item["Age"].N))
}

func (s FixturesDynamoDbSuite) TestDynamoDbKvStore() {
	loader := fixtures.NewFixtureLoader(dynamoDbKvStoreFixtures())

	config := configFromFiles("test_configs/config.dynamodb.test.yml", "test_configs/config.fixtures_dynamodb.test.yml")

	err := loader.Load(config, s.logger)
	assert.NoError(s.T(), err)

	gio, err := s.db.GetItem(&dynamodb.GetItemInput{
		Key: map[string]*dynamodb.AttributeValue{
			"key": {
				S: mdl.String("Ash"),
			},
		},
		TableName: mdl.String("gosoline-test-integration-test-test-application-kvstore-testModel"),
	})

	// should have created the item
	assert.NoError(s.T(), err)
	assert.Len(s.T(), gio.Item, 2, "2 attributes expected")
	assert.JSONEq(s.T(), `{"Name":"Ash","Age":10}`, *(gio.Item["value"].S))
}

type DynamoDbTestModel struct {
	Name string `ddb:"key=hash"`
	Age  uint
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
				&fixtures.KvstoreFixture{
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
			Writer: fixtures.DynamoDbFixtureWriterFactory(&mdl.ModelId{
				Project:     "gosoline",
				Environment: "test",
				Family:      "integration-test",
				Application: "test-application",
				Name:        "testModel",
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
