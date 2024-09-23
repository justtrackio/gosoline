//go:build integration && fixtures
// +build integration,fixtures

package dynamodb_test

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	gosoAws "github.com/justtrackio/gosoline/pkg/cloud/aws"
	"github.com/justtrackio/gosoline/pkg/ddb"
	"github.com/justtrackio/gosoline/pkg/fixtures"
	"github.com/justtrackio/gosoline/pkg/mdl"
	"github.com/justtrackio/gosoline/pkg/test/suite"
)

type DynamoDbSuite struct {
	suite.Suite
}

func (s *DynamoDbSuite) SetupSuite() []suite.Option {
	err := os.Setenv("AWS_ACCESS_KEY_ID", gosoAws.DefaultAccessKeyID)
	s.NoError(err)

	err = os.Setenv("AWS_SECRET_ACCESS_KEY", gosoAws.DefaultSecretAccessKey)
	s.NoError(err)

	return []suite.Option{
		suite.WithLogLevel("debug"),
		suite.WithConfigFile("config.test.yml"),
		suite.WithSharedEnvironment(),
	}
}

func (s *DynamoDbSuite) TestDynamoDb() {
	envContext := s.Env().Context()
	envConfig := s.Env().Config()
	envLogger := s.Env().Logger()

	loader := fixtures.NewFixtureLoader(envContext, envConfig, envLogger)

	fss, err := s.dynamoDbFixtureSet1()
	s.NoError(err)

	err = loader.Load(envContext, "default", fss)
	s.NoError(err)

	ddbClient := s.Env().DynamoDb("default").Client()

	gio, err := ddbClient.GetItem(context.Background(), &dynamodb.GetItemInput{
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

	qo, err := ddbClient.Query(context.Background(), &dynamodb.QueryInput{
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

	_, err = ddbClient.DeleteTable(envContext, &dynamodb.DeleteTableInput{TableName: aws.String("gosoline-test-integration-test-grp-testModel")})
	s.NoError(err)
}

func (s *DynamoDbSuite) TestDynamoDbWithPurge() {
	envContext := s.Env().Context()
	envConfig := s.Env().Config()
	envLogger := s.Env().Logger()

	loader := fixtures.NewFixtureLoader(envContext, envConfig, envLogger)

	fss, err := s.dynamoDbFixtureSet1()
	s.NoError(err)

	err = loader.Load(envContext, "default", fss)
	s.NoError(err)

	ddbClient := s.Env().DynamoDb("default").Client()

	gio, err := ddbClient.GetItem(context.Background(), &dynamodb.GetItemInput{
		Key: map[string]types.AttributeValue{
			"Name": &types.AttributeValueMemberS{
				Value: "Ash",
			},
		},
		TableName: aws.String("gosoline-test-integration-test-grp-testModel"),
	})

	// should have created the first item
	s.NoError(err)
	s.Len(gio.Item, 2, "2 attributes expected")
	s.Equal("Ash", gio.Item["Name"].(*types.AttributeValueMemberS).Value)
	s.Equal("10", gio.Item["Age"].(*types.AttributeValueMemberN).Value)

	fss, err = s.dynamodbFixtureSet2()
	s.NoError(err)

	err = loader.Load(envContext, "default", fss)
	s.NoError(err)

	gio, err = ddbClient.GetItem(context.Background(), &dynamodb.GetItemInput{
		Key: map[string]types.AttributeValue{
			"Name": &types.AttributeValueMemberS{
				Value: "Bash",
			},
		},
		TableName: aws.String("gosoline-test-integration-test-grp-testModel"),
	})

	// should have created the second item
	s.NoError(err)
	s.Len(gio.Item, 2, "2 attributes expected")
	s.Equal("Bash", gio.Item["Name"].(*types.AttributeValueMemberS).Value)
	s.Equal("10", gio.Item["Age"].(*types.AttributeValueMemberN).Value)

	gio, err = ddbClient.GetItem(context.Background(), &dynamodb.GetItemInput{
		Key: map[string]types.AttributeValue{
			"Name": &types.AttributeValueMemberS{
				Value: "Ash",
			},
		},
		TableName: aws.String("gosoline-test-integration-test-grp-testModel"),
	})

	// first item should have been purged
	s.NoError(err)
	s.Nil(gio.Item)

	qo, err := ddbClient.Query(context.Background(), &dynamodb.QueryInput{
		TableName:              aws.String("gosoline-test-integration-test-grp-testModel"),
		IndexName:              aws.String("IDX_Age"),
		KeyConditionExpression: aws.String("Age = :v_age"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":v_age": &types.AttributeValueMemberN{
				Value: "10",
			},
		},
	})

	s.NoError(err)
	s.Len(qo.Items, 1, "1 item expected")

	_, err = ddbClient.DeleteTable(envContext, &dynamodb.DeleteTableInput{TableName: aws.String("gosoline-test-integration-test-grp-testModel")})
	s.NoError(err)
}

func (s *DynamoDbSuite) TestDynamoDbKvStore() {
	envContext := s.Env().Context()
	envConfig := s.Env().Config()
	envLogger := s.Env().Logger()

	loader := fixtures.NewFixtureLoader(envContext, envConfig, envLogger)

	fss, err := s.dynamoDbKvStoreFixtureSet1()
	s.NoError(err)

	err = loader.Load(envContext, "default", fss)
	s.NoError(err)

	ddbClient := s.Env().DynamoDb("default").Client()

	gio, err := ddbClient.GetItem(context.Background(), &dynamodb.GetItemInput{
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

	_, err = ddbClient.DeleteTable(envContext, &dynamodb.DeleteTableInput{TableName: aws.String("gosoline-test-integration-test-grp-kvstore-testModel")})
	s.NoError(err)
}

func (s *DynamoDbSuite) TestDynamoDbKvStoreWithPurge() {
	envContext := s.Env().Context()
	envConfig := s.Env().Config()
	envLogger := s.Env().Logger()

	loader := fixtures.NewFixtureLoader(envContext, envConfig, envLogger)

	fss, err := s.dynamoDbKvStoreFixtureSet1()
	s.NoError(err)

	err = loader.Load(envContext, "default", fss)
	s.NoError(err)

	ddbClient := s.Env().DynamoDb("default").Client()

	gio, err := ddbClient.GetItem(context.Background(), &dynamodb.GetItemInput{
		Key: map[string]types.AttributeValue{
			"key": &types.AttributeValueMemberS{
				Value: "Ash",
			},
		},
		TableName: aws.String("gosoline-test-integration-test-grp-kvstore-testModel"),
	})

	// should have created the first item
	s.NoError(err)
	s.Len(gio.Item, 2, "2 attributes expected")

	expectedValue := &types.AttributeValueMemberS{
		Value: `{"Name":"Ash","Age":10}`,
	}
	s.Equal(expectedValue, gio.Item["value"].(*types.AttributeValueMemberS))

	fss, err = s.dynamoDbKvStoreFixtureSet2()
	s.NoError(err)

	err = loader.Load(envContext, "default", fss)
	s.NoError(err)

	gio, err = ddbClient.GetItem(context.Background(), &dynamodb.GetItemInput{
		Key: map[string]types.AttributeValue{
			"key": &types.AttributeValueMemberS{
				Value: "Bash",
			},
		},
		TableName: aws.String("gosoline-test-integration-test-grp-kvstore-testModel"),
	})

	// should have created the second item
	s.NoError(err)
	s.Len(gio.Item, 2, "2 attributes expected")

	expectedKey := &types.AttributeValueMemberS{
		Value: "Bash",
	}
	s.Equal(expectedKey, gio.Item["key"].(*types.AttributeValueMemberS))

	expectedValue = &types.AttributeValueMemberS{
		Value: `{"Name":"Bash","Age":10}`,
	}
	s.Equal(expectedValue, gio.Item["value"].(*types.AttributeValueMemberS))

	gio, err = ddbClient.GetItem(context.Background(), &dynamodb.GetItemInput{
		Key: map[string]types.AttributeValue{
			"key": &types.AttributeValueMemberS{
				Value: "Ash",
			},
		},
		TableName: aws.String("gosoline-test-integration-test-grp-kvstore-testModel"),
	})

	// first item should have been purged
	s.NoError(err)
	s.Nil(gio.Item)

	_, err = ddbClient.DeleteTable(envContext, &dynamodb.DeleteTableInput{TableName: aws.String("gosoline-test-integration-test-grp-kvstore-testModel")})
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

func (s *DynamoDbSuite) dynamoDbFixtureSets(data fixtures.NamedFixtures[*Person], purge bool) ([]fixtures.FixtureSet, error) {
	writer, err := fixtures.NewDynamoDbFixtureWriter(s.Env().Context(), s.Env().Config(), s.Env().Logger(), ddbSettings)
	if err != nil {
		return nil, fmt.Errorf("failed to create ddb fixture writer: %w", err)
	}

	return []fixtures.FixtureSet{fixtures.NewSimpleFixtureSet(data, writer, fixtures.WithPurge(purge))}, nil
}

func (s *DynamoDbSuite) dynamoDbFixtureSet1() ([]fixtures.FixtureSet, error) {
	return s.dynamoDbFixtureSets(fixtures.NamedFixtures[*Person]{
		&fixtures.NamedFixture[*Person]{
			Name:  "ash",
			Value: &Person{Name: "Ash", Age: 10},
		},
	}, false)
}

func (s *DynamoDbSuite) dynamodbFixtureSet2() ([]fixtures.FixtureSet, error) {
	return s.dynamoDbFixtureSets(fixtures.NamedFixtures[*Person]{
		&fixtures.NamedFixture[*Person]{
			Name:  "bash",
			Value: &Person{Name: "Bash", Age: 10},
		},
	}, true)
}

func (s *DynamoDbSuite) dynamoDbKvStoreFixtures(data fixtures.NamedFixtures[*fixtures.KvStoreFixture], purge bool) ([]fixtures.FixtureSet, error) {
	writer, err := fixtures.NewDynamoDbKvStoreFixtureWriter[Person](s.Env().Context(), s.Env().Config(), s.Env().Logger(), kvStoreSettings)
	if err != nil {
		return nil, fmt.Errorf("failed to create ddb fixture writer: %w", err)
	}

	return []fixtures.FixtureSet{
		fixtures.NewSimpleFixtureSet(data, writer, fixtures.WithPurge(purge)),
	}, nil
}

func (s *DynamoDbSuite) dynamoDbKvStoreFixtureSet1() ([]fixtures.FixtureSet, error) {
	return s.dynamoDbKvStoreFixtures(fixtures.NamedFixtures[*fixtures.KvStoreFixture]{
		&fixtures.NamedFixture[*fixtures.KvStoreFixture]{
			Name: "ash",
			Value: &fixtures.KvStoreFixture{
				Key:   "Ash",
				Value: Person{Name: "Ash", Age: 10},
			},
		},
	}, false)
}

func (s *DynamoDbSuite) dynamoDbKvStoreFixtureSet2() ([]fixtures.FixtureSet, error) {
	return s.dynamoDbKvStoreFixtures(fixtures.NamedFixtures[*fixtures.KvStoreFixture]{
		&fixtures.NamedFixture[*fixtures.KvStoreFixture]{
			Name: "bash",
			Value: &fixtures.KvStoreFixture{
				Key:   "Bash",
				Value: Person{Name: "Bash", Age: 10},
			},
		},
	}, true)
}

func TestDynamoDbSuite(t *testing.T) {
	suite.Run(t, new(DynamoDbSuite))
}
