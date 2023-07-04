package ddb_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/smithy-go"
	dynamodbMocks "github.com/justtrackio/gosoline/pkg/cloud/aws/dynamodb/mocks"
	"github.com/justtrackio/gosoline/pkg/ddb"
	"github.com/justtrackio/gosoline/pkg/exec"
	logMocks "github.com/justtrackio/gosoline/pkg/log/mocks"
	"github.com/justtrackio/gosoline/pkg/mdl"
	"github.com/justtrackio/gosoline/pkg/tracing"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

type model struct {
	Id  int    `json:"id" ddb:"key=hash"`
	Rev string `json:"rev" ddb:"key=range"`
	Foo string `json:"foo"`
}

type projection struct {
	Id int `json:"id"`
}

type RepositoryTestSuite struct {
	suite.Suite
	ctx    context.Context
	client *dynamodbMocks.Client
	repo   ddb.Repository
}

func (s *RepositoryTestSuite) SetupTest() {
	logger := logMocks.NewLoggerMockedAll()
	tracer := tracing.NewNoopTracer()

	s.ctx = context.Background()
	s.client = dynamodbMocks.NewClient(s.T())

	tableSettings := &ddb.Settings{
		ModelId: mdl.ModelId{
			Project:     "applike",
			Environment: "test",
			Family:      "gosoline",
			Application: "ddb",
			Name:        "myModel",
		},
		Main: ddb.MainSettings{
			Model: model{},
		},
	}

	metadataFactory := ddb.NewMetadataFactoryWithInterfaces(tableSettings, "applike-test-gosoline-ddb-myModel")

	var err error
	s.repo, err = ddb.NewWithInterfaces(logger, tracer, s.client, metadataFactory)
	s.NoError(err)
}

func (s *RepositoryTestSuite) TestGetItem() {
	item := model{}
	input := &dynamodb.GetItemInput{
		Key: map[string]types.AttributeValue{
			"id":  &types.AttributeValueMemberN{Value: "1"},
			"rev": &types.AttributeValueMemberS{Value: "0"},
		},
		TableName:              aws.String("applike-test-gosoline-ddb-myModel"),
		ReturnConsumedCapacity: types.ReturnConsumedCapacityIndexes,
	}
	output := &dynamodb.GetItemOutput{
		ConsumedCapacity: &types.ConsumedCapacity{},
		Item: map[string]types.AttributeValue{
			"id":  &types.AttributeValueMemberN{Value: "1"},
			"rev": &types.AttributeValueMemberS{Value: "0"},
			"foo": &types.AttributeValueMemberS{Value: "bar"},
		},
	}

	s.client.EXPECT().GetItem(mock.AnythingOfType("*context.valueCtx"), input).Return(output, nil)

	qb := s.repo.GetItemBuilder().WithHash(1).WithRange("0")
	res, err := s.repo.GetItem(s.ctx, qb, &item)

	expected := model{
		Id:  1,
		Rev: "0",
		Foo: "bar",
	}

	s.NoError(err)
	s.True(res.IsFound)
	s.EqualValues(expected, item)
}

func (s *RepositoryTestSuite) TestGetItem_FromItem() {
	input := &dynamodb.GetItemInput{
		Key: map[string]types.AttributeValue{
			"id":  &types.AttributeValueMemberN{Value: "5"},
			"rev": &types.AttributeValueMemberS{Value: "abc"},
		},
		TableName:              aws.String("applike-test-gosoline-ddb-myModel"),
		ReturnConsumedCapacity: types.ReturnConsumedCapacityIndexes,
	}
	output := &dynamodb.GetItemOutput{
		ConsumedCapacity: &types.ConsumedCapacity{},
		Item: map[string]types.AttributeValue{
			"id":  &types.AttributeValueMemberN{Value: "5"},
			"rev": &types.AttributeValueMemberS{Value: "abc"},
			"foo": &types.AttributeValueMemberS{Value: "baz"},
		},
	}

	s.client.EXPECT().GetItem(mock.AnythingOfType("*context.valueCtx"), input).Return(output, nil)

	item := model{
		Id:  5,
		Rev: "abc",
	}

	qb := s.repo.GetItemBuilder().WithHash(5).WithRange("abc")
	res, err := s.repo.GetItem(context.Background(), qb, &item)

	expected := model{
		Id:  5,
		Rev: "abc",
		Foo: "baz",
	}

	s.NoError(err)
	s.True(res.IsFound)
	s.EqualValues(expected, item)
}

func (s *RepositoryTestSuite) TestGetItemNotFound() {
	item := model{}

	input := &dynamodb.GetItemInput{
		Key: map[string]types.AttributeValue{
			"id":  &types.AttributeValueMemberN{Value: "1"},
			"rev": &types.AttributeValueMemberS{Value: "0"},
		},
		TableName:              aws.String("applike-test-gosoline-ddb-myModel"),
		ReturnConsumedCapacity: types.ReturnConsumedCapacityIndexes,
	}
	output := &dynamodb.GetItemOutput{
		ConsumedCapacity: &types.ConsumedCapacity{},
	}

	s.client.EXPECT().GetItem(mock.AnythingOfType("*context.valueCtx"), input).Return(output, nil)

	qb := s.repo.GetItemBuilder().WithHash(1).WithRange("0")
	res, err := s.repo.GetItem(s.ctx, qb, &item)

	s.NoError(err)
	s.False(res.IsFound)
}

func (s *RepositoryTestSuite) TestGetItemProjection() {
	input := &dynamodb.GetItemInput{
		ExpressionAttributeNames: map[string]string{
			"#0": "id",
		},
		Key: map[string]types.AttributeValue{
			"id":  &types.AttributeValueMemberN{Value: "1"},
			"rev": &types.AttributeValueMemberS{Value: "0"},
		},
		ProjectionExpression:   aws.String("#0"),
		TableName:              aws.String("applike-test-gosoline-ddb-myModel"),
		ReturnConsumedCapacity: types.ReturnConsumedCapacityIndexes,
	}
	output := &dynamodb.GetItemOutput{
		ConsumedCapacity: &types.ConsumedCapacity{},
		Item: map[string]types.AttributeValue{
			"id": &types.AttributeValueMemberN{Value: "1"},
		},
	}

	s.client.EXPECT().GetItem(mock.AnythingOfType("*context.valueCtx"), input).Return(output, nil)

	item := projection{}

	qb := s.repo.GetItemBuilder().WithHash(1).WithRange("0").WithProjection(item)
	res, err := s.repo.GetItem(s.ctx, qb, &item)

	expected := projection{
		Id: 1,
	}

	s.NoError(err)
	s.True(res.IsFound)
	s.EqualValues(expected, item)
}

func (s *RepositoryTestSuite) TestQuery() {
	input := &dynamodb.QueryInput{
		ExpressionAttributeNames: map[string]string{
			"#0": "id",
		},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":0": &types.AttributeValueMemberN{Value: "1"},
		},
		KeyConditionExpression: aws.String("#0 = :0"),
		TableName:              aws.String("applike-test-gosoline-ddb-myModel"),
		ReturnConsumedCapacity: types.ReturnConsumedCapacityIndexes,
	}
	output := &dynamodb.QueryOutput{
		ConsumedCapacity: &types.ConsumedCapacity{},
		Count:            2,
		ScannedCount:     2,
		Items: []map[string]types.AttributeValue{
			{
				"id":  &types.AttributeValueMemberN{Value: "1"},
				"rev": &types.AttributeValueMemberS{Value: "0"},
				"foo": &types.AttributeValueMemberS{Value: "bar"},
			},
			{
				"id":  &types.AttributeValueMemberN{Value: "1"},
				"rev": &types.AttributeValueMemberS{Value: "1"},
				"foo": &types.AttributeValueMemberS{Value: "baz"},
			},
		},
	}

	s.client.EXPECT().Query(mock.AnythingOfType("*context.valueCtx"), input).Return(output, nil)

	result := make([]model, 0)
	expected := []model{
		{
			Id:  1,
			Rev: "0",
			Foo: "bar",
		},
		{
			Id:  1,
			Rev: "1",
			Foo: "baz",
		},
	}

	qb := s.repo.QueryBuilder().WithHash(1)
	_, err := s.repo.Query(s.ctx, qb, &result)

	s.NoError(err)
	s.Len(result, 2)
	s.EqualValues(expected, result)
}

func (s *RepositoryTestSuite) TestQuery_Canceled() {
	awsErr := &smithy.CanceledError{}

	input := &dynamodb.QueryInput{
		TableName:              aws.String("applike-test-gosoline-ddb-myModel"),
		KeyConditionExpression: aws.String("#0 = :0"),
		ExpressionAttributeNames: map[string]string{
			"#0": "id",
		},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":0": &types.AttributeValueMemberN{Value: "1"},
		},
		ReturnConsumedCapacity: types.ReturnConsumedCapacityIndexes,
	}

	s.client.EXPECT().Query(mock.AnythingOfType("*context.valueCtx"), input).Return(nil, awsErr)

	result := make([]model, 0)

	qb := s.repo.QueryBuilder().WithHash(1)
	_, err := s.repo.Query(s.ctx, qb, &result)

	s.Error(err)

	isRequestCanceled := errors.Is(err, exec.RequestCanceledError)
	s.True(isRequestCanceled)
}

func (s *RepositoryTestSuite) TestBatchGetItems() {
	input := &dynamodb.BatchGetItemInput{
		RequestItems: map[string]types.KeysAndAttributes{
			"applike-test-gosoline-ddb-myModel": {
				Keys: []map[string]types.AttributeValue{
					{
						"id":  &types.AttributeValueMemberN{Value: "1"},
						"rev": &types.AttributeValueMemberS{Value: "0"},
					},
					{
						"id":  &types.AttributeValueMemberN{Value: "2"},
						"rev": &types.AttributeValueMemberS{Value: "0"},
					},
				},
			},
		},
		ReturnConsumedCapacity: types.ReturnConsumedCapacityIndexes,
	}
	output := &dynamodb.BatchGetItemOutput{
		Responses: map[string][]map[string]types.AttributeValue{
			"applike-test-gosoline-ddb-myModel": {
				{
					"id":  &types.AttributeValueMemberN{Value: "1"},
					"rev": &types.AttributeValueMemberS{Value: "0"},
					"foo": &types.AttributeValueMemberS{Value: "foo"},
				},
				{
					"id":  &types.AttributeValueMemberN{Value: "2"},
					"rev": &types.AttributeValueMemberS{Value: "0"},
					"foo": &types.AttributeValueMemberS{Value: "bar"},
				},
			},
		},
		UnprocessedKeys: map[string]types.KeysAndAttributes{},
	}

	s.client.EXPECT().BatchGetItem(mock.AnythingOfType("*context.valueCtx"), input).Return(output, nil)

	result := make([]model, 0)
	expected := []model{
		{
			Id:  1,
			Rev: "0",
			Foo: "foo",
		},
		{
			Id:  2,
			Rev: "0",
			Foo: "bar",
		},
	}

	qb := s.repo.BatchGetItemsBuilder().WithKeys(1, "0").WithKeys(2, "0")
	_, err := s.repo.BatchGetItems(s.ctx, qb, &result)

	s.NoError(err)
	s.Equal(expected, result)
}

func (s *RepositoryTestSuite) TestBatchWriteItem() {
	items := []model{
		{
			Id:  1,
			Rev: "0",
			Foo: "foo",
		},
		{
			Id:  2,
			Rev: "0",
			Foo: "bar",
		},
	}

	input := &dynamodb.BatchWriteItemInput{
		RequestItems: map[string][]types.WriteRequest{
			"applike-test-gosoline-ddb-myModel": {
				{
					PutRequest: &types.PutRequest{
						Item: map[string]types.AttributeValue{
							"id":  &types.AttributeValueMemberN{Value: "1"},
							"rev": &types.AttributeValueMemberS{Value: "0"},
							"foo": &types.AttributeValueMemberS{Value: "foo"},
						},
					},
				},
				{
					PutRequest: &types.PutRequest{
						Item: map[string]types.AttributeValue{
							"id":  &types.AttributeValueMemberN{Value: "2"},
							"rev": &types.AttributeValueMemberS{Value: "0"},
							"foo": &types.AttributeValueMemberS{Value: "bar"},
						},
					},
				},
			},
		},
		ReturnConsumedCapacity: types.ReturnConsumedCapacityIndexes,
	}

	output := &dynamodb.BatchWriteItemOutput{
		UnprocessedItems: map[string][]types.WriteRequest{},
	}

	s.client.EXPECT().BatchWriteItem(mock.AnythingOfType("*context.valueCtx"), input).Return(output, nil)

	_, err := s.repo.BatchPutItems(s.ctx, items)

	s.NoError(err)
}

func (s *RepositoryTestSuite) TestBatchWriteItem_Retry() {
	makeItem := func(id int) model {
		return model{
			Id:  id,
			Rev: fmt.Sprintf("rev %d", id),
			Foo: "data",
		}
	}
	makePutRequest := func(id int) *types.PutRequest {
		return &types.PutRequest{
			Item: map[string]types.AttributeValue{
				"id":  &types.AttributeValueMemberN{Value: fmt.Sprintf("%d", id)},
				"rev": &types.AttributeValueMemberS{Value: fmt.Sprintf("rev %d", id)},
				"foo": &types.AttributeValueMemberS{Value: "data"},
			},
		}
	}

	totalItems := 20
	firstBatchItems := 10

	items := make([]model, 0, totalItems)
	firstInputData := make([]types.WriteRequest, 0, totalItems)
	firstOutputData := make([]types.WriteRequest, 0, firstBatchItems)
	secondInputData := make([]types.WriteRequest, 0, firstBatchItems)

	for i := 0; i < totalItems; i++ {
		items = append(items, makeItem(i))

		firstInputData = append(firstInputData, types.WriteRequest{
			PutRequest: makePutRequest(i),
		})

		if i < firstBatchItems {
			secondInputData = append(secondInputData, types.WriteRequest{
				PutRequest: makePutRequest(i),
			})
			firstOutputData = append(firstOutputData, types.WriteRequest{
				PutRequest: makePutRequest(i),
			})
		}
	}

	firstInput := &dynamodb.BatchWriteItemInput{
		RequestItems: map[string][]types.WriteRequest{
			"applike-test-gosoline-ddb-myModel": firstInputData,
		},
		ReturnConsumedCapacity: types.ReturnConsumedCapacityIndexes,
	}
	secondInput := &dynamodb.BatchWriteItemInput{
		RequestItems: map[string][]types.WriteRequest{
			"applike-test-gosoline-ddb-myModel": secondInputData,
		},
		ReturnConsumedCapacity: types.ReturnConsumedCapacityIndexes,
	}

	firstOutput := &dynamodb.BatchWriteItemOutput{
		UnprocessedItems: map[string][]types.WriteRequest{
			"applike-test-gosoline-ddb-myModel": firstOutputData,
		},
	}
	secondOutput := &dynamodb.BatchWriteItemOutput{
		UnprocessedItems: map[string][]types.WriteRequest{},
	}

	s.client.EXPECT().BatchWriteItem(mock.AnythingOfType("*context.valueCtx"), firstInput).Return(firstOutput, nil).Once()
	s.client.EXPECT().BatchWriteItem(mock.AnythingOfType("*context.valueCtx"), secondInput).Return(secondOutput, nil).Once()

	_, err := s.repo.BatchPutItems(s.ctx, items)

	s.NoError(err)
}

func (s *RepositoryTestSuite) TestPutItem() {
	item := model{
		Id:  1,
		Rev: "0",
		Foo: "foo",
	}

	input := &dynamodb.PutItemInput{
		TableName: aws.String("applike-test-gosoline-ddb-myModel"),
		Item: map[string]types.AttributeValue{
			"id":  &types.AttributeValueMemberN{Value: "1"},
			"rev": &types.AttributeValueMemberS{Value: "0"},
			"foo": &types.AttributeValueMemberS{Value: "foo"},
		},
		ReturnConsumedCapacity: types.ReturnConsumedCapacityIndexes,
	}
	output := &dynamodb.PutItemOutput{
		ConsumedCapacity: &types.ConsumedCapacity{},
	}

	s.client.EXPECT().PutItem(mock.AnythingOfType("*context.valueCtx"), input).Return(output, nil)

	res, err := s.repo.PutItem(s.ctx, nil, item)

	s.NoError(err)
	s.False(res.ConditionalCheckFailed)
}

func (s *RepositoryTestSuite) TestUpdate() {
	input := &dynamodb.UpdateItemInput{
		TableName: aws.String("applike-test-gosoline-ddb-myModel"),
		Key: map[string]types.AttributeValue{
			"id":  &types.AttributeValueMemberN{Value: "1"},
			"rev": &types.AttributeValueMemberS{Value: "0"},
		},
		ExpressionAttributeNames: map[string]string{
			"#0": "foo",
		},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":0": &types.AttributeValueMemberS{Value: "bar"},
		},
		UpdateExpression:       aws.String("SET #0 = :0\n"),
		ReturnValues:           types.ReturnValueAllNew,
		ReturnConsumedCapacity: types.ReturnConsumedCapacityIndexes,
	}
	output := &dynamodb.UpdateItemOutput{
		ConsumedCapacity: &types.ConsumedCapacity{},
		Attributes: map[string]types.AttributeValue{
			"id":  &types.AttributeValueMemberN{Value: "1"},
			"rev": &types.AttributeValueMemberS{Value: "0"},
			"foo": &types.AttributeValueMemberS{Value: "bar"},
		},
	}

	s.client.EXPECT().UpdateItem(mock.AnythingOfType("*context.valueCtx"), input).Return(output, nil)

	updatedItem := &model{
		Id:  1,
		Rev: "0",
	}
	ub := s.repo.UpdateItemBuilder().Set("foo", "bar").ReturnAllNew()
	res, err := s.repo.UpdateItem(s.ctx, ub, updatedItem)

	expectedItem := &model{
		Id:  1,
		Rev: "0",
		Foo: "bar",
	}

	s.NoError(err)
	s.False(res.ConditionalCheckFailed)
	s.EqualValues(expectedItem, updatedItem)
}

func (s *RepositoryTestSuite) TestDeleteItem() {
	input := &dynamodb.DeleteItemInput{
		ConditionExpression: aws.String("#0 = :0"),
		ExpressionAttributeNames: map[string]string{
			"#0": "foo",
		},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":0": &types.AttributeValueMemberS{Value: "bar"},
		},
		Key: map[string]types.AttributeValue{
			"id":  &types.AttributeValueMemberN{Value: "1"},
			"rev": &types.AttributeValueMemberS{Value: "0"},
		},
		ReturnValues:           types.ReturnValueAllOld,
		ReturnConsumedCapacity: types.ReturnConsumedCapacityIndexes,
		TableName:              aws.String("applike-test-gosoline-ddb-myModel"),
	}
	output := &dynamodb.DeleteItemOutput{
		ConsumedCapacity: &types.ConsumedCapacity{},
		Attributes: map[string]types.AttributeValue{
			"id":  &types.AttributeValueMemberN{Value: "1"},
			"rev": &types.AttributeValueMemberS{Value: "0"},
			"foo": &types.AttributeValueMemberS{Value: "bar"},
		},
	}

	s.client.EXPECT().DeleteItem(mock.AnythingOfType("*context.valueCtx"), input).Return(output, nil)

	item := model{
		Id:  1,
		Rev: "0",
		Foo: "baz",
	}

	expected := model{
		Id:  1,
		Rev: "0",
		Foo: "bar",
	}

	db := s.repo.DeleteItemBuilder().WithCondition(ddb.Eq("foo", "bar")).ReturnAllOld()
	res, err := s.repo.DeleteItem(s.ctx, db, &item)

	s.NoError(err)
	s.False(res.ConditionalCheckFailed)
	s.Equal(expected, item)
}

func TestRepositoryTestSuite(t *testing.T) {
	suite.Run(t, new(RepositoryTestSuite))
}
