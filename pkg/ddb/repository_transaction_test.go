package ddb_test

import (
	"context"
	"errors"
	"strconv"
	"testing"

	dynamodbMocks "github.com/applike/gosoline/pkg/cloud/aws/dynamodb/mocks"
	"github.com/applike/gosoline/pkg/ddb"
	ddbMocks "github.com/applike/gosoline/pkg/ddb/mocks"
	logMocks "github.com/applike/gosoline/pkg/log/mocks"
	tracingMocks "github.com/applike/gosoline/pkg/tracing/mocks"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type RepositoryTransactionTestSuite struct {
	suite.Suite

	ctx        context.Context
	span       *tracingMocks.Span
	logger     *logMocks.Logger
	client     *dynamodbMocks.Client
	tracer     *tracingMocks.Tracer
	repository ddb.TransactionRepository
}

func TestRepositoryTransactionTestSuite(t *testing.T) {
	suite.Run(t, new(RepositoryTransactionTestSuite))
}

func (s *RepositoryTransactionTestSuite) SetupTest() {
	s.ctx = context.Background()
	s.logger = logMocks.NewLoggerMockedAll()
	s.client = new(dynamodbMocks.Client)
	s.tracer = new(tracingMocks.Tracer)
	s.span = new(tracingMocks.Span)

	s.repository = ddb.NewTransactionRepositoryWithInterfaces(s.logger, s.client, s.tracer)
}

func (s *RepositoryTransactionTestSuite) TearDownTest() {
	s.span.AssertExpectations(s.T())
	s.client.AssertExpectations(s.T())
	s.tracer.AssertExpectations(s.T())
}

func (s *RepositoryTransactionTestSuite) TestTransactGetItems() {
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
		ConsumedCapacity: []types.ConsumedCapacity{{
			TableName:         aws.String("model"),
			CapacityUnits:     aws.Float64(6),
			ReadCapacityUnits: aws.Float64(6),
		}},
		Responses: []types.ItemResponse{
			{
				Item: map[string]types.AttributeValue{
					"id":  &types.AttributeValueMemberN{Value: "1"},
					"rev": &types.AttributeValueMemberS{Value: "1"},
					"foo": &types.AttributeValueMemberS{Value: "1"},
				},
			},
			{
				Item: map[string]types.AttributeValue{
					"id":  &types.AttributeValueMemberN{Value: "2"},
					"rev": &types.AttributeValueMemberS{Value: "2"},
					"foo": &types.AttributeValueMemberS{Value: "2"},
				},
			},
			{
				Item: map[string]types.AttributeValue{
					"id":  &types.AttributeValueMemberN{Value: "3"},
					"rev": &types.AttributeValueMemberS{Value: "3"},
					"foo": &types.AttributeValueMemberS{Value: "3"},
				},
			},
		},
	}

	s.tracer.
		On("StartSubSpan", s.ctx, "ddb.TransactGetItems").
		Return(s.ctx, s.span)

	s.span.
		On("Finish").
		Return()

	s.client.On("TransactGetItems", s.ctx, mock.AnythingOfType("*dynamodb.TransactGetItemsInput")).Return(requestOutput, nil)

	result, err := s.repository.TransactGetItems(s.ctx, getEntries)

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
		Return(&types.ConditionCheck{
			ConditionExpression: aws.String("#foo = :bar"),
			ExpressionAttributeNames: map[string]string{
				"#foo": "foo",
			},
			ExpressionAttributeValues: map[string]types.AttributeValue{
				":bar": &types.AttributeValueMemberS{Value: "bar"},
			},
			Key: map[string]types.AttributeValue{
				"#id":  &types.AttributeValueMemberN{Value: "42"},
				"#rev": &types.AttributeValueMemberN{Value: "foo"},
			},
			ReturnValuesOnConditionCheckFailure: types.ReturnValuesOnConditionCheckFailureAllOld,
			TableName:                           aws.String("model"),
		}, nil)

	items := []ddb.TransactWriteItemBuilder{
		&ddb.TransactConditionCheck{
			Builder: conditionCheckBuilder,
			Item:    conditionCheckItem,
		},
	}

	s.tracer.
		On("StartSubSpan", s.ctx, "ddb.TransactWriteItems").
		Return(s.ctx, s.span)

	s.span.
		On("Finish").
		Return()

	requestErr := &types.TransactionCanceledException{
		CancellationReasons: []types.CancellationReason{
			{
				Code: aws.String("ConditionalCheckFailed"),
				Item: map[string]types.AttributeValue{
					"id":  &types.AttributeValueMemberN{Value: "42"},
					"rev": &types.AttributeValueMemberS{Value: "foo"},
					"foo": &types.AttributeValueMemberS{Value: "foo"},
				},
			},
		},
	}

	s.client.On("TransactWriteItems", s.ctx, mock.AnythingOfType("*dynamodb.TransactWriteItemsInput")).Return(nil, requestErr)

	result, err := s.repository.TransactWriteItems(s.ctx, items)

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
			ExpressionAttributeNames: map[string]string{
				"#id":  "id",
				"#rev": "rev",
				"#foo": "foo",
			},
			ExpressionAttributeValues: map[string]types.AttributeValue{
				"#id":  &types.AttributeValueMemberN{Value: "42"},
				"#rev": &types.AttributeValueMemberN{Value: "foo"},
				"#foo": &types.AttributeValueMemberN{Value: "bar"},
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
			ExpressionAttributeNames: map[string]string{
				"foo": "foo",
			},
			ExpressionAttributeValues: map[string]types.AttributeValue{
				"bar": &types.AttributeValueMemberS{Value: "bar"},
			},
			Key: map[string]types.AttributeValue{
				"id":  &types.AttributeValueMemberN{Value: "24"},
				"rev": &types.AttributeValueMemberS{Value: "foo"},
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

	requestOutput := &dynamodb.TransactWriteItemsOutput{ConsumedCapacity: []types.ConsumedCapacity{{
		CapacityUnits:      aws.Float64(4),
		ReadCapacityUnits:  aws.Float64(2),
		WriteCapacityUnits: aws.Float64(2),
	}}}

	s.client.On("TransactWriteItems", s.ctx, mock.AnythingOfType("*dynamodb.TransactWriteItemsInput")).Return(requestOutput, nil)

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
		Key: map[string]types.AttributeValue{
			"id":  &types.AttributeValueMemberN{Value: strconv.Itoa(item.Id)},
			"rev": &types.AttributeValueMemberS{Value: item.Rev},
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
