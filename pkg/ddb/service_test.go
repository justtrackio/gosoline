package ddb_test

import (
	"context"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	dynamodbMocks "github.com/justtrackio/gosoline/pkg/cloud/aws/dynamodb/mocks"
	"github.com/justtrackio/gosoline/pkg/ddb"
	logMocks "github.com/justtrackio/gosoline/pkg/log/mocks"
	"github.com/justtrackio/gosoline/pkg/mdl"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type createModel struct {
	Id        int       `json:"id" ddb:"key=hash"`
	Rev       string    `json:"rev" ddb:"key=range"`
	Name      string    `json:"name" ddb:"global=hash"`
	Header    string    `json:"header"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"createdAt" ddb:"local=range"`
	UpdatedAt time.Time `json:"updatedAt" ddb:"global=range"`
	Ttl       int       `json:"ttl" ddb:"ttl=enabled"`
}

type secondaryModel1 struct {
	Id   int    `json:"id" ddb:"key=hash"`
	Body string `json:"body" ddb:"local=range"`
}

type secondaryModel2 struct {
	Id        int       `json:"id" ddb:"key=hash"`
	Name      string    `json:"name"`
	UpdatedAt time.Time `json:"updatedAt" ddb:"local=range"`
}

type globalModel1 struct {
	Rev       string    `json:"rev" ddb:"global=hash"`
	CreatedAt time.Time `json:"createdAt" ddb:"global=range"`
	Header    string    `json:"header"`
}

func TestService_CreateTable(t *testing.T) {
	ctx := context.Background()
	logger := logMocks.NewLoggerMockedAll()
	client := new(dynamodbMocks.Client)

	describeCount := 0
	describeInput := &dynamodb.DescribeTableInput{
		TableName: aws.String("applike-test-gosoline-ddb-myModel"),
	}
	describeOutput := &dynamodb.DescribeTableOutput{
		Table: &types.TableDescription{
			TableStatus: types.TableStatusActive,
		},
	}
	client.On("DescribeTable", ctx, describeInput).Run(func(args mock.Arguments) {
		describeCount++
	}).Return(func(_ context.Context, _ *dynamodb.DescribeTableInput, _ ...func(options *dynamodb.Options)) *dynamodb.DescribeTableOutput {
		if describeCount == 0 {
			return nil
		}

		return describeOutput
	}, func(_ context.Context, _ *dynamodb.DescribeTableInput, _ ...func(options *dynamodb.Options)) error {
		if describeCount == 0 {
			return &types.ResourceNotFoundException{}
		}

		return nil
	})

	createInput := &dynamodb.CreateTableInput{
		AttributeDefinitions: []types.AttributeDefinition{
			{
				AttributeName: aws.String("body"),
				AttributeType: types.ScalarAttributeTypeS,
			},
			{
				AttributeName: aws.String("createdAt"),
				AttributeType: types.ScalarAttributeTypeS,
			},
			{
				AttributeName: aws.String("id"),
				AttributeType: types.ScalarAttributeTypeN,
			},
			{
				AttributeName: aws.String("name"),
				AttributeType: types.ScalarAttributeTypeS,
			},
			{
				AttributeName: aws.String("rev"),
				AttributeType: types.ScalarAttributeTypeS,
			},
			{
				AttributeName: aws.String("updatedAt"),
				AttributeType: types.ScalarAttributeTypeS,
			},
		},
		GlobalSecondaryIndexes: []types.GlobalSecondaryIndex{
			{
				IndexName: aws.String("foo-index"),
				KeySchema: []types.KeySchemaElement{
					{
						AttributeName: aws.String("rev"),
						KeyType:       types.KeyTypeHash,
					}, {
						AttributeName: aws.String("createdAt"),
						KeyType:       types.KeyTypeRange,
					},
				},
				Projection: &types.Projection{
					NonKeyAttributes: []string{"header"},
					ProjectionType:   types.ProjectionTypeInclude,
				},
				ProvisionedThroughput: &types.ProvisionedThroughput{
					ReadCapacityUnits:  aws.Int64(7),
					WriteCapacityUnits: aws.Int64(8),
				},
			},
			{
				IndexName: aws.String("global-name"),
				KeySchema: []types.KeySchemaElement{
					{
						AttributeName: aws.String("name"),
						KeyType:       types.KeyTypeHash,
					}, {
						AttributeName: aws.String("updatedAt"),
						KeyType:       types.KeyTypeRange,
					},
				},
				Projection: &types.Projection{
					NonKeyAttributes: nil,
					ProjectionType:   types.ProjectionTypeAll,
				},
				ProvisionedThroughput: &types.ProvisionedThroughput{
					ReadCapacityUnits:  aws.Int64(4),
					WriteCapacityUnits: aws.Int64(5),
				},
			},
		},
		KeySchema: []types.KeySchemaElement{
			{
				AttributeName: aws.String("id"),
				KeyType:       types.KeyTypeHash,
			}, {
				AttributeName: aws.String("rev"),
				KeyType:       types.KeyTypeRange,
			},
		},
		LocalSecondaryIndexes: []types.LocalSecondaryIndex{
			{
				IndexName: aws.String("local-body"),
				KeySchema: []types.KeySchemaElement{
					{
						AttributeName: aws.String("id"),
						KeyType:       types.KeyTypeHash,
					}, {
						AttributeName: aws.String("body"),
						KeyType:       types.KeyTypeRange,
					},
				},
				Projection: &types.Projection{
					NonKeyAttributes: nil,
					ProjectionType:   types.ProjectionTypeKeysOnly,
				},
			},
			{
				IndexName: aws.String("local-createdAt"),
				KeySchema: []types.KeySchemaElement{
					{
						AttributeName: aws.String("id"),
						KeyType:       types.KeyTypeHash,
					}, {
						AttributeName: aws.String("createdAt"),
						KeyType:       types.KeyTypeRange,
					},
				},
				Projection: &types.Projection{
					NonKeyAttributes: nil,
					ProjectionType:   types.ProjectionTypeAll,
				},
			},
			{
				IndexName: aws.String("local-updatedAt"),
				KeySchema: []types.KeySchemaElement{
					{
						AttributeName: aws.String("id"),
						KeyType:       types.KeyTypeHash,
					}, {
						AttributeName: aws.String("updatedAt"),
						KeyType:       types.KeyTypeRange,
					},
				},
				Projection: &types.Projection{
					NonKeyAttributes: []string{"name"},
					ProjectionType:   types.ProjectionTypeInclude,
				},
			},
		},
		ProvisionedThroughput: &types.ProvisionedThroughput{
			ReadCapacityUnits:  aws.Int64(1),
			WriteCapacityUnits: aws.Int64(2),
		},
		StreamSpecification: &types.StreamSpecification{
			StreamEnabled:  aws.Bool(true),
			StreamViewType: types.StreamViewTypeNewImage,
		},
		TableName: aws.String("applike-test-gosoline-ddb-myModel"),
	}
	client.On("CreateTable", ctx, createInput).Return(nil, nil)

	ttlInput := &dynamodb.UpdateTimeToLiveInput{
		TableName: aws.String("applike-test-gosoline-ddb-myModel"),
		TimeToLiveSpecification: &types.TimeToLiveSpecification{
			AttributeName: aws.String("ttl"),
			Enabled:       aws.Bool(true),
		},
	}
	client.On("UpdateTimeToLive", ctx, ttlInput).Return(nil, nil)

	settings := &ddb.Settings{
		ModelId: mdl.ModelId{
			Project:     "applike",
			Environment: "test",
			Family:      "gosoline",
			Application: "ddb",
			Name:        "myModel",
		},
		AutoCreate: true,
		Main: ddb.MainSettings{
			Model:              createModel{},
			StreamView:         ddb.StreamViewTypeNewImage,
			ReadCapacityUnits:  1,
			WriteCapacityUnits: 2,
		},
		Local: []ddb.LocalSettings{
			{
				Model: createModel{},
			},
			{
				Model: secondaryModel1{},
			},
			{
				Model: secondaryModel2{},
			},
		},
		Global: []ddb.GlobalSettings{
			{
				Model:              createModel{},
				ReadCapacityUnits:  4,
				WriteCapacityUnits: 5,
			},
			{
				Name:               "foo-index",
				Model:              globalModel1{},
				ReadCapacityUnits:  7,
				WriteCapacityUnits: 8,
			},
		},
	}

	metadataFactory := ddb.NewMetadataFactoryWithInterfaces(settings, "applike-test-gosoline-ddb-myModel")
	svc := ddb.NewServiceWithInterfaces(logger, client, metadataFactory)

	_, err := svc.CreateTable(ctx)

	assert.NoError(t, err)
}
