package ddb_test

import (
	"github.com/applike/gosoline/pkg/cloud/mocks"
	"github.com/applike/gosoline/pkg/ddb"
	logMocks "github.com/applike/gosoline/pkg/log/mocks"
	"github.com/applike/gosoline/pkg/mdl"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"testing"
	"time"
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
	logger := logMocks.NewLoggerMockedAll()
	client := new(mocks.DynamoDBAPI)

	describeCount := 0
	describeInput := &dynamodb.DescribeTableInput{
		TableName: aws.String("applike-test-gosoline-ddb-myModel"),
	}
	describeOutput := &dynamodb.DescribeTableOutput{
		Table: &dynamodb.TableDescription{
			TableStatus: aws.String(dynamodb.TableStatusActive),
		},
	}
	client.On("DescribeTable", describeInput).Run(func(args mock.Arguments) {
		describeCount++
	}).Return(func(_ *dynamodb.DescribeTableInput) *dynamodb.DescribeTableOutput {
		if describeCount == 0 {
			return nil
		}

		return describeOutput
	}, func(_ *dynamodb.DescribeTableInput) error {
		if describeCount == 0 {
			return awserr.New(dynamodb.ErrCodeResourceNotFoundException, "", nil)
		}

		return nil
	})

	createInput := &dynamodb.CreateTableInput{
		AttributeDefinitions: []*dynamodb.AttributeDefinition{
			{
				AttributeName: aws.String("body"),
				AttributeType: aws.String(dynamodb.ScalarAttributeTypeS),
			},
			{
				AttributeName: aws.String("createdAt"),
				AttributeType: aws.String(dynamodb.ScalarAttributeTypeS),
			},
			{
				AttributeName: aws.String("id"),
				AttributeType: aws.String(dynamodb.ScalarAttributeTypeN),
			},
			{
				AttributeName: aws.String("name"),
				AttributeType: aws.String(dynamodb.ScalarAttributeTypeS),
			},
			{
				AttributeName: aws.String("rev"),
				AttributeType: aws.String(dynamodb.ScalarAttributeTypeS),
			},
			{
				AttributeName: aws.String("updatedAt"),
				AttributeType: aws.String(dynamodb.ScalarAttributeTypeS),
			},
		},
		GlobalSecondaryIndexes: []*dynamodb.GlobalSecondaryIndex{
			{
				IndexName: aws.String("foo-index"),
				KeySchema: []*dynamodb.KeySchemaElement{
					{
						AttributeName: aws.String("rev"),
						KeyType:       aws.String(dynamodb.KeyTypeHash),
					}, {
						AttributeName: aws.String("createdAt"),
						KeyType:       aws.String(dynamodb.KeyTypeRange),
					},
				},
				Projection: &dynamodb.Projection{
					NonKeyAttributes: []*string{aws.String("header")},
					ProjectionType:   aws.String(dynamodb.ProjectionTypeInclude),
				},
				ProvisionedThroughput: &dynamodb.ProvisionedThroughput{
					ReadCapacityUnits:  aws.Int64(7),
					WriteCapacityUnits: aws.Int64(8),
				},
			},
			{
				IndexName: aws.String("global-name"),
				KeySchema: []*dynamodb.KeySchemaElement{
					{
						AttributeName: aws.String("name"),
						KeyType:       aws.String(dynamodb.KeyTypeHash),
					}, {
						AttributeName: aws.String("updatedAt"),
						KeyType:       aws.String(dynamodb.KeyTypeRange),
					},
				},
				Projection: &dynamodb.Projection{
					NonKeyAttributes: nil,
					ProjectionType:   aws.String(dynamodb.ProjectionTypeAll),
				},
				ProvisionedThroughput: &dynamodb.ProvisionedThroughput{
					ReadCapacityUnits:  aws.Int64(4),
					WriteCapacityUnits: aws.Int64(5),
				},
			},
		},
		KeySchema: []*dynamodb.KeySchemaElement{
			{
				AttributeName: aws.String("id"),
				KeyType:       aws.String(dynamodb.KeyTypeHash),
			}, {
				AttributeName: aws.String("rev"),
				KeyType:       aws.String(dynamodb.KeyTypeRange),
			},
		},
		LocalSecondaryIndexes: []*dynamodb.LocalSecondaryIndex{
			{
				IndexName: aws.String("local-body"),
				KeySchema: []*dynamodb.KeySchemaElement{
					{
						AttributeName: aws.String("id"),
						KeyType:       aws.String(dynamodb.KeyTypeHash),
					}, {
						AttributeName: aws.String("body"),
						KeyType:       aws.String(dynamodb.KeyTypeRange),
					},
				},
				Projection: &dynamodb.Projection{
					NonKeyAttributes: nil,
					ProjectionType:   aws.String(dynamodb.ProjectionTypeKeysOnly),
				},
			},
			{
				IndexName: aws.String("local-createdAt"),
				KeySchema: []*dynamodb.KeySchemaElement{
					{
						AttributeName: aws.String("id"),
						KeyType:       aws.String(dynamodb.KeyTypeHash),
					}, {
						AttributeName: aws.String("createdAt"),
						KeyType:       aws.String(dynamodb.KeyTypeRange),
					},
				},
				Projection: &dynamodb.Projection{
					NonKeyAttributes: nil,
					ProjectionType:   aws.String(dynamodb.ProjectionTypeAll),
				},
			},
			{
				IndexName: aws.String("local-updatedAt"),
				KeySchema: []*dynamodb.KeySchemaElement{
					{
						AttributeName: aws.String("id"),
						KeyType:       aws.String(dynamodb.KeyTypeHash),
					}, {
						AttributeName: aws.String("updatedAt"),
						KeyType:       aws.String(dynamodb.KeyTypeRange),
					},
				},
				Projection: &dynamodb.Projection{
					NonKeyAttributes: []*string{aws.String("name")},
					ProjectionType:   aws.String(dynamodb.ProjectionTypeInclude),
				},
			},
		},
		ProvisionedThroughput: &dynamodb.ProvisionedThroughput{
			ReadCapacityUnits:  aws.Int64(1),
			WriteCapacityUnits: aws.Int64(2),
		},
		StreamSpecification: &dynamodb.StreamSpecification{
			StreamEnabled:  aws.Bool(true),
			StreamViewType: aws.String(dynamodb.StreamViewTypeNewImage),
		},
		TableName: aws.String("applike-test-gosoline-ddb-myModel"),
	}
	client.On("CreateTable", createInput).Return(nil, nil)

	ttlInput := &dynamodb.UpdateTimeToLiveInput{
		TableName: aws.String("applike-test-gosoline-ddb-myModel"),
		TimeToLiveSpecification: &dynamodb.TimeToLiveSpecification{
			AttributeName: aws.String("ttl"),
			Enabled:       aws.Bool(true),
		},
	}
	client.On("UpdateTimeToLive", ttlInput).Return(nil, nil)

	svc := ddb.NewServiceWithInterfaces(logger, client)

	_, err := svc.CreateTable(&ddb.Settings{
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
	})

	assert.NoError(t, err)
}
