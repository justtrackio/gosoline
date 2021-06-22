package ddb_test

import (
	"context"
	"errors"
	cloudAws "github.com/applike/gosoline/pkg/cloud/aws"
	cloudMocks "github.com/applike/gosoline/pkg/cloud/mocks"
	"github.com/applike/gosoline/pkg/ddb"
	ddbMocks "github.com/applike/gosoline/pkg/ddb/mocks"
	logMocks "github.com/applike/gosoline/pkg/log/mocks"
	tracingMocks "github.com/applike/gosoline/pkg/tracing/mocks"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"strconv"
	"testing"
)

type RepositoryTransactionTestSuite struct {
	suite.Suite

	span     *tracingMocks.Span
	logger   *logMocks.Logger
	client   *cloudMocks.DynamoDBAPI
	executor *cloudAws.TestableExecutor
	tracer   *tracingMocks.Tracer

	repository ddb.TransactionRepository
}

func TestRepositoryTransactionTestSuite(t *testing.T) {
	suite.Run(t, new(RepositoryTransactionTestSuite))
}

func (s *RepositoryTransactionTestSuite) SetupTest() {
	s.logger = logMocks.NewLoggerMockedAll()
	s.client = new(cloudMocks.DynamoDBAPI)
	s.tracer = new(tracingMocks.Tracer)
	s.span = new(tracingMocks.Span)

	s.executor = cloudAws.NewTestableExecutor(&s.client.Mock)

	s.repository = ddb.NewTransactionRepositoryWithInterfaces(s.logger, s.client, s.executor, s.tracer)
}

func (s *RepositoryTransactionTestSuite) TearDownTest() {
	s.span.AssertExpectations(s.T())
	s.client.AssertExpectations(s.T())
	s.executor.AssertExpectations(s.T())
	s.tracer.AssertExpectations(s.T())
}

func (s *RepositoryTransactionTestSuite) TestTransactGetItems() {
	ctx := context.Background()

	models := []model{
		{
			Id:  1,
			Rev: "1",
		}, {
			Id:  2,
			Rev: "2",
		}, {
			Id:  3,
			Rev: "3",
		},
	}

	getEntries := make([]ddb.TransactGetItemBuilder, len(models))
	for i := range models {
		getEntries[i] = buildTransactGetItemBuilder(&models[i])
	}

	requestOutput := &dynamodb.TransactGetItemsOutput{
		ConsumedCapacity: []*dynamodb.ConsumedCapacity{{
			TableName:         aws.String("model"),
			CapacityUnits:     aws.Float64(6),
			ReadCapacityUnits: aws.Float64(6),
		}},
		Responses: []*dynamodb.ItemResponse{
			{
				Item: map[string]*dynamodb.AttributeValue{
					"id": {
						N: aws.String("1"),
					},
					"rev": {
						S: aws.String("1"),
					},
					"foo": {
						S: aws.String("1"),
					},
				},
			},
			{
				Item: map[string]*dynamodb.AttributeValue{
					"id": {
						N: aws.String("2"),
					},
					"rev": {
						S: aws.String("2"),
					},
					"foo": {
						S: aws.String("2"),
					},
				},
			},
			{
				Item: map[string]*dynamodb.AttributeValue{
					"id": {
						N: aws.String("3"),
					},
					"rev": {
						S: aws.String("3"),
					},
					"foo": {
						S: aws.String("3"),
					},
				},
			},
		},
	}

	s.tracer.
		On("StartSubSpan", ctx, "ddb.TransactGetItems").
		Return(ctx, s.span)

	s.span.
		On("Finish").
		Return()

	s.executor.ExpectExecution("TransactGetItemsRequest", mock.AnythingOfType("*dynamodb.TransactGetItemsInput"), requestOutput, nil)

	result, err := s.repository.TransactGetItems(ctx, getEntries)

	require.NoError(s.T(), err)
	require.NotNil(s.T(), result)

	expectedResult := &ddb.OperationResult{ConsumedCapacity: &ddb.ConsumedCapacity{
		Total: 6,
		Read:  6,
		LSI:   make(map[string]*ddb.Capacity),
		GSI:   make(map[string]*ddb.Capacity),
		Table: &ddb.Capacity{},
	}}

	expectedModels := []model{
		{
			Id:  1,
			Rev: "1",
			Foo: "1",
		}, {
			Id:  2,
			Rev: "2",
			Foo: "2",
		}, {
			Id:  3,
			Rev: "3",
			Foo: "3",
		},
	}

	assert.Equal(s.T(), expectedResult, result)
	assert.Equal(s.T(), expectedModels, models)
}

func (s *RepositoryTransactionTestSuite) TestTransactWriteItems_ConditionCheckFailed() {
	conditionCheckItem := &model{
		Id:  42,
		Rev: "foo",
	}

	conditionCheckBuilder := new(ddbMocks.ConditionCheckBuilder)
	conditionCheckBuilder.
		On("Build", conditionCheckItem).
		Return(&dynamodb.ConditionCheck{
			ConditionExpression: aws.String("#foo = :bar"),
			ExpressionAttributeNames: map[string]*string{
				"#foo": aws.String("foo"),
			},
			ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
				":bar": {
					S: aws.String("bar"),
				},
			},
			Key: map[string]*dynamodb.AttributeValue{
				"#id": {
					N: aws.String("42"),
				},
				"#rev": {
					N: aws.String("foo"),
				},
			},
			ReturnValuesOnConditionCheckFailure: aws.String(dynamodb.ReturnValueAllOld),
			TableName:                           aws.String("model"),
		}, nil)

	ctx := context.Background()

	items := []ddb.TransactWriteItemBuilder{
		&ddb.TransactConditionCheck{
			Builder: conditionCheckBuilder,
			Item:    conditionCheckItem,
		},
	}

	s.tracer.
		On("StartSubSpan", ctx, "ddb.TransactWriteItems").
		Return(ctx, s.span)

	s.span.
		On("Finish").
		Return()

	requestErr := &dynamodb.TransactionCanceledException{
		CancellationReasons: []*dynamodb.CancellationReason{
			{
				Code: aws.String("ConditionalCheckFailed"),
				Item: map[string]*dynamodb.AttributeValue{
					"id": {
						N: aws.String("42"),
					},
					"rev": {
						S: aws.String("foo"),
					},
					"foo": {
						S: aws.String("foo"),
					},
				},
			},
		},
	}

	s.executor.
		ExpectExecution("TransactWriteItemsRequest", mock.AnythingOfType("*dynamodb.TransactWriteItemsInput"), nil, requestErr)

	result, err := s.repository.TransactWriteItems(ctx, items)

	require.Nil(s.T(), result)
	require.Error(s.T(), err)
	require.True(s.T(), errors.Is(err, ddb.ErrorConditionalCheckFailed))

	expectedItem := &model{
		Id:  42,
		Rev: "foo",
		Foo: "foo",
	}

	assert.Equal(s.T(), expectedItem, conditionCheckItem)
}

func (s *RepositoryTransactionTestSuite) TestTransactWriteItems() {
	putItem := &model{
		Id:  42,
		Rev: "foo",
		Foo: "bar",
	}

	putItemBuilder := new(ddbMocks.PutItemBuilder)
	putItemBuilder.
		On("Build", putItem).
		Return(&dynamodb.PutItemInput{
			ConditionExpression: aws.String("attribute_not_exists(id)"),
			ExpressionAttributeNames: map[string]*string{
				"#id":  aws.String("id"),
				"#rev": aws.String("rev"),
				"#foo": aws.String("foo"),
			},
			ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
				"#id": {
					N: aws.String("42"),
				},
				"#rev": {
					N: aws.String("foo"),
				},
				"#foo": {
					N: aws.String("bar"),
				},
			},
			TableName: aws.String("model"),
		}, nil)

	updateItem := &model{
		Id:  24, // must not operate on the same item twice within one transaction
		Rev: "foo",
	}

	updateItemBuilder := new(ddbMocks.UpdateItemBuilder)
	updateItemBuilder.
		On("Build", updateItem).
		Return(&dynamodb.UpdateItemInput{
			ConditionExpression: aws.String(""),
			ExpressionAttributeNames: map[string]*string{
				"foo": aws.String("foo"),
			},
			ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
				"bar": {
					S: aws.String("bar"),
				},
			},
			Key: map[string]*dynamodb.AttributeValue{
				"id": {
					N: aws.String("24"),
				},
				"rev": {
					S: aws.String("foo"),
				},
			},
			TableName:        aws.String("model"),
			UpdateExpression: aws.String("#foo = :bar"),
		}, nil)

	ctx := context.Background()

	items := []ddb.TransactWriteItemBuilder{
		&ddb.TransactPutItem{
			Builder: putItemBuilder,
			Item:    putItem,
		},
		&ddb.TransactUpdateItem{
			Builder: updateItemBuilder,
			Item:    updateItem,
		},
	}

	s.tracer.
		On("StartSubSpan", ctx, "ddb.TransactWriteItems").
		Return(ctx, s.span)

	s.span.
		On("Finish").
		Return()

	requestOutput := &dynamodb.TransactWriteItemsOutput{ConsumedCapacity: []*dynamodb.ConsumedCapacity{{
		CapacityUnits:      aws.Float64(4),
		ReadCapacityUnits:  aws.Float64(2),
		WriteCapacityUnits: aws.Float64(2),
	}}}

	s.executor.
		ExpectExecution("TransactWriteItemsRequest", mock.AnythingOfType("*dynamodb.TransactWriteItemsInput"), requestOutput, nil)

	result, err := s.repository.TransactWriteItems(ctx, items)

	require.NoError(s.T(), err)
	require.NotNil(s.T(), result)

	expected := &ddb.OperationResult{ConsumedCapacity: &ddb.ConsumedCapacity{
		Total: 4,
		Read:  2,
		Write: 2,
		LSI:   make(map[string]*ddb.Capacity),
		GSI:   make(map[string]*ddb.Capacity),
		Table: &ddb.Capacity{},
	}}

	assert.Equal(s.T(), expected, result)
}

func buildTransactGetItemBuilder(item *model) ddb.TransactGetItemBuilder {
	builder := new(ddbMocks.GetItemBuilder)

	input := &dynamodb.GetItemInput{
		ConsistentRead: aws.Bool(false),
		Key: map[string]*dynamodb.AttributeValue{
			"id": {
				N: aws.String(strconv.Itoa(item.Id)),
			},
			"rev": {
				S: aws.String(item.Rev),
			},
		},
		TableName: aws.String("model"),
	}

	builder.
		On("Build", item).
		Return(input, nil)

	return &ddb.TransactGetItem{
		Builder: builder,
		Item:    item,
	}
}
