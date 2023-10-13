package ddb_test

import (
	"context"
	"strconv"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	dynamodbMocks "github.com/justtrackio/gosoline/pkg/cloud/aws/dynamodb/mocks"
	"github.com/justtrackio/gosoline/pkg/ddb"
	ddbMocks "github.com/justtrackio/gosoline/pkg/ddb/mocks"
	logMocks "github.com/justtrackio/gosoline/pkg/log/mocks"
	tracingMocks "github.com/justtrackio/gosoline/pkg/tracing/mocks"
	"github.com/stretchr/testify/mock"
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
	s.client = dynamodbMocks.NewClient(s.T())
	s.tracer = tracingMocks.NewTracer(s.T())
	s.span = tracingMocks.NewSpan(s.T())

	s.repository = ddb.NewTransactionRepositoryWithInterfaces(s.logger, s.client, s.tracer)
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
		getEntries[i] = s.buildTransactGetItemBuilder(&models[i])
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

	s.client.EXPECT().TransactGetItems(context.Background(), mock.AnythingOfType("*dynamodb.TransactGetItemsInput")).Return(requestOutput, nil)

	result, err := s.repository.TransactGetItems(s.ctx, getEntries)

	s.NoError(err)
	s.NotNil(result)

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

	s.Equal(6.0, result.ConsumedCapacity.Read())
	s.Equal(6.0, result.ConsumedCapacity.Total())
	s.Equal(expectedModels, models)
}

func (s *RepositoryTransactionTestSuite) TestTransactWriteItems() {
	putItem := &model{
		Id:  42,
		Rev: "foo",
		Foo: "bar",
	}

	putItemBuilder := ddbMocks.NewPutItemBuilder(s.T())
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

	updateItemBuilder := ddbMocks.NewUpdateItemBuilder(s.T())
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
		ReadCapacityUnits:  aws.Float64(3),
		WriteCapacityUnits: aws.Float64(1),
	}}}

	s.client.EXPECT().TransactWriteItems(context.Background(), mock.AnythingOfType("*dynamodb.TransactWriteItemsInput")).Return(requestOutput, nil)

	result, err := s.repository.TransactWriteItems(ctx, items)

	s.NoError(err)
	s.NotNil(result)

	s.Equal(4.0, result.ConsumedCapacity.Total())
	s.Equal(3.0, result.ConsumedCapacity.Read())
	s.Equal(1.0, result.ConsumedCapacity.Write())
}

func (s *RepositoryTransactionTestSuite) buildTransactGetItemBuilder(item *model) ddb.TransactGetItemBuilder {
	builder := ddbMocks.NewGetItemBuilder(s.T())

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
