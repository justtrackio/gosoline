//go:build integration && fixtures

package dynamodb_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/ddb"
	"github.com/justtrackio/gosoline/pkg/fixtures"
	"github.com/justtrackio/gosoline/pkg/kvstore"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/mdl"
	"github.com/justtrackio/gosoline/pkg/test/suite"
)

func TestDynamoDbSuite(t *testing.T) {
	suite.Run(t, new(DynamoDbSuite))
}

type DynamoDbSuite struct {
	suite.Suite
}

func (s *DynamoDbSuite) SetupSuite() []suite.Option {
	return []suite.Option{
		suite.WithLogLevel("debug"),
		suite.WithConfigFile("config.test.yml"),
		suite.WithSharedEnvironment(),
	}
}

func (s *DynamoDbSuite) TestDynamoDb() {
	err := s.Env().LoadFixtureSets(s.dynamoDbFixtureSet1())
	s.NoError(err)

	ddbClient := s.Env().Localstack("default").DdbClient()
	gio, err := ddbClient.GetItem(s.T().Context(), &dynamodb.GetItemInput{
		Key: map[string]types.AttributeValue{
			"Name": &types.AttributeValueMemberS{
				Value: "Ash",
			},
		},
		TableName: aws.String("gosoline-test-integration-test-grp-testModel"),
	})

	// should have created the item
	s.NoError(err)
	s.Len(gio.Item, 2, "2 attributes expected")
	s.Equal("Ash", gio.Item["Name"].(*types.AttributeValueMemberS).Value)
	s.Equal("10", gio.Item["Age"].(*types.AttributeValueMemberN).Value)

	qo, err := ddbClient.Query(s.T().Context(), &dynamodb.QueryInput{
		TableName:              aws.String("gosoline-test-integration-test-grp-testModel"),
		IndexName:              aws.String("IDX_Age"),
		KeyConditionExpression: aws.String("Age = :v_age"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":v_age": &types.AttributeValueMemberN{
				Value: "10",
			},
		},
	})

	// should have created global index
	s.NoError(err)
	s.Len(qo.Items, 1, "1 item expected")

	_, err = ddbClient.DeleteTable(s.T().Context(), &dynamodb.DeleteTableInput{TableName: aws.String("gosoline-test-integration-test-grp-testModel")})
	s.NoError(err)
}

func (s *DynamoDbSuite) TestDynamoDbKvStore() {
	err := s.Env().LoadFixtureSets(s.dynamoDbKvStoreFixtureSet1())
	s.NoError(err)

	ddbClient := s.Env().Localstack("default").DdbClient()
	gio, err := ddbClient.GetItem(s.T().Context(), &dynamodb.GetItemInput{
		Key: map[string]types.AttributeValue{
			"key": &types.AttributeValueMemberS{
				Value: "Ash",
			},
		},
		TableName: aws.String("gosoline-test-integration-test-grp-kvstore-testModel"),
	})

	// should have created the item
	s.NoError(err)
	s.Len(gio.Item, 2, "2 attributes expected")

	expectedKey := &types.AttributeValueMemberS{
		Value: "Ash",
	}
	s.Equal(expectedKey, gio.Item["key"].(*types.AttributeValueMemberS))

	expectedValue := &types.AttributeValueMemberS{
		Value: `{"Name":"Ash","Age":10}`,
	}
	s.Equal(expectedValue, gio.Item["value"].(*types.AttributeValueMemberS))

	_, err = ddbClient.DeleteTable(s.T().Context(), &dynamodb.DeleteTableInput{TableName: aws.String("gosoline-test-integration-test-grp-kvstore-testModel")})
	s.NoError(err)
}

type Person struct {
	Name string `ddb:"key=hash"`
	Age  uint   `ddb:"global=hash"`
}

var kvStoreSettings = &mdl.ModelId{
	Project:     "gosoline",
	Environment: "test",
	Family:      "integration-test",
	Group:       "grp",
	Application: "test-application",
	Name:        "testModel",
}

var ddbSettings = &ddb.Settings{
	ModelId: mdl.ModelId{
		Project:     "gosoline",
		Environment: "test",
		Family:      "integration-test",
		Group:       "grp",
		Application: "test-application",
		Name:        "testModel",
	},
	Main: ddb.MainSettings{
		Model: Person{},
	},
	Global: []ddb.GlobalSettings{
		{
			Name:               "IDX_Age",
			Model:              Person{},
			ReadCapacityUnits:  1,
			WriteCapacityUnits: 1,
		},
	},
}

func (s *DynamoDbSuite) dynamoDbFixtureSets(data fixtures.NamedFixtures[*Person]) []fixtures.FixtureSetsFactory {
	return []fixtures.FixtureSetsFactory{
		func(ctx context.Context, config cfg.Config, logger log.Logger, group string) ([]fixtures.FixtureSet, error) {
			writer, err := ddb.NewDynamoDbFixtureWriter(s.Env().Context(), s.Env().Config(), s.Env().Logger(), ddbSettings)
			if err != nil {
				return nil, fmt.Errorf("failed to create ddb fixture writer: %w", err)
			}

			return []fixtures.FixtureSet{fixtures.NewSimpleFixtureSet(data, writer)}, nil
		},
	}
}

func (s *DynamoDbSuite) dynamoDbFixtureSet1() []fixtures.FixtureSetsFactory {
	return s.dynamoDbFixtureSets(fixtures.NamedFixtures[*Person]{
		&fixtures.NamedFixture[*Person]{
			Name:  "ash",
			Value: &Person{Name: "Ash", Age: 10},
		},
	})
}

func (s *DynamoDbSuite) dynamoDbKvStoreFixtures(data fixtures.NamedFixtures[*kvstore.KvStoreFixture]) []fixtures.FixtureSetsFactory {
	return []fixtures.FixtureSetsFactory{
		func(ctx context.Context, config cfg.Config, logger log.Logger, group string) ([]fixtures.FixtureSet, error) {
			writer, err := kvstore.NewDynamoDbKvStoreFixtureWriter[Person](s.Env().Context(), s.Env().Config(), s.Env().Logger(), kvStoreSettings)
			if err != nil {
				return nil, fmt.Errorf("failed to create ddb fixture writer: %w", err)
			}

			return []fixtures.FixtureSet{
				fixtures.NewSimpleFixtureSet(data, writer),
			}, nil
		},
	}
}

func (s *DynamoDbSuite) dynamoDbKvStoreFixtureSet1() []fixtures.FixtureSetsFactory {
	return s.dynamoDbKvStoreFixtures(fixtures.NamedFixtures[*kvstore.KvStoreFixture]{
		&fixtures.NamedFixture[*kvstore.KvStoreFixture]{
			Name: "ash",
			Value: &kvstore.KvStoreFixture{
				Key:   "Ash",
				Value: Person{Name: "Ash", Age: 10},
			},
		},
	})
}
