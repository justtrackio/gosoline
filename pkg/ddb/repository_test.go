package ddb_test

import (
	"context"
	"errors"
	"fmt"
	gosoAws "github.com/applike/gosoline/pkg/cloud/aws"
	cloudMocks "github.com/applike/gosoline/pkg/cloud/mocks"
	"github.com/applike/gosoline/pkg/ddb"
	"github.com/applike/gosoline/pkg/exec"
	logMocks "github.com/applike/gosoline/pkg/log/mocks"
	"github.com/applike/gosoline/pkg/mdl"
	"github.com/applike/gosoline/pkg/tracing"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/stretchr/testify/suite"
	"strconv"
	"testing"
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
	executor *gosoAws.TestableExecutor
	repo     ddb.Repository
}

func (s *RepositoryTestSuite) SetupTest() {
	logger := logMocks.NewLoggerMockedAll()
	tracer := tracing.NewNoopTracer()
	client := new(cloudMocks.DynamoDBAPI)
	s.executor = gosoAws.NewTestableExecutor(&client.Mock)

	var err error
	s.repo, err = ddb.NewWithInterfaces(logger, tracer, client, s.executor, &ddb.Settings{
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
	})
	s.NoError(err)
}

func (s *RepositoryTestSuite) TestGetItem() {
	item := model{}
	input := &dynamodb.GetItemInput{
		Key: map[string]*dynamodb.AttributeValue{
			"id": {
				N: aws.String("1"),
			},
			"rev": {
				S: aws.String("0"),
			},
		},
		TableName: aws.String("applike-test-gosoline-ddb-myModel"),
	}
	output := &dynamodb.GetItemOutput{
		ConsumedCapacity: nil,
		Item: map[string]*dynamodb.AttributeValue{
			"id": {
				N: aws.String(strconv.Itoa(1)),
			},
			"rev": {
				S: aws.String("0"),
			},
			"foo": {
				S: aws.String("bar"),
			},
		},
	}

	s.executor.ExpectExecution("GetItemRequest", input, output, nil)

	qb := s.repo.GetItemBuilder().WithHash(1).WithRange("0")
	res, err := s.repo.GetItem(context.Background(), qb, &item)

	expected := model{
		Id:  1,
		Rev: "0",
		Foo: "bar",
	}

	s.NoError(err)
	s.True(res.IsFound)
	s.EqualValues(expected, item)

	s.executor.AssertExpectations(s.T())
}

func (s *RepositoryTestSuite) TestGetItem_FromItem() {
	input := &dynamodb.GetItemInput{
		Key: map[string]*dynamodb.AttributeValue{
			"id": {
				N: aws.String("5"),
			},
			"rev": {
				S: aws.String("abc"),
			},
		},
		TableName: aws.String("applike-test-gosoline-ddb-myModel"),
	}
	output := &dynamodb.GetItemOutput{
		ConsumedCapacity: nil,
		Item: map[string]*dynamodb.AttributeValue{
			"id": {
				N: aws.String("5"),
			},
			"rev": {
				S: aws.String("abc"),
			},
			"foo": {
				S: aws.String("baz"),
			},
		},
	}

	s.executor.ExpectExecution("GetItemRequest", input, output, nil)

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
		Key: map[string]*dynamodb.AttributeValue{
			"id": {
				N: aws.String(strconv.Itoa(1)),
			},
			"rev": {
				S: aws.String("0"),
			},
		},
		TableName: aws.String("applike-test-gosoline-ddb-myModel"),
	}
	output := &dynamodb.GetItemOutput{}

	s.executor.ExpectExecution("GetItemRequest", input, output, nil)

	qb := s.repo.GetItemBuilder().WithHash(1).WithRange("0")
	res, err := s.repo.GetItem(context.Background(), qb, &item)

	s.NoError(err)
	s.False(res.IsFound)

	s.executor.AssertExpectations(s.T())
}

func (s *RepositoryTestSuite) TestGetItemProjection() {
	input := &dynamodb.GetItemInput{
		ExpressionAttributeNames: map[string]*string{
			"#0": aws.String("id"),
		},
		Key: map[string]*dynamodb.AttributeValue{
			"id": {
				N: aws.String(strconv.Itoa(1)),
			},
			"rev": {
				S: aws.String("0"),
			},
		},
		ProjectionExpression: aws.String("#0"),
		TableName:            aws.String("applike-test-gosoline-ddb-myModel"),
	}
	output := &dynamodb.GetItemOutput{
		ConsumedCapacity: nil,
		Item: map[string]*dynamodb.AttributeValue{
			"id": {
				N: aws.String(strconv.Itoa(1)),
			},
		},
	}

	s.executor.ExpectExecution("GetItemRequest", input, output, nil)

	item := projection{}

	qb := s.repo.GetItemBuilder().WithHash(1).WithRange("0").WithProjection(item)
	res, err := s.repo.GetItem(context.Background(), qb, &item)

	expected := projection{
		Id: 1,
	}

	s.NoError(err)
	s.True(res.IsFound)
	s.EqualValues(expected, item)

	s.executor.AssertExpectations(s.T())
}

func (s *RepositoryTestSuite) TestQuery() {
	input := &dynamodb.QueryInput{
		ExpressionAttributeNames: map[string]*string{
			"#0": aws.String("id"),
		},
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":0": {
				N: aws.String("1"),
			},
		},
		KeyConditionExpression: aws.String("#0 = :0"),
		TableName:              aws.String("applike-test-gosoline-ddb-myModel"),
	}
	output := &dynamodb.QueryOutput{
		Count:        aws.Int64(2),
		ScannedCount: aws.Int64(2),
		Items: []map[string]*dynamodb.AttributeValue{
			{
				"id": {
					N: aws.String("1"),
				},
				"rev": {
					S: aws.String("0"),
				},
				"foo": {
					S: aws.String("bar"),
				},
			},
			{
				"id": {
					N: aws.String("1"),
				},
				"rev": {
					S: aws.String("1"),
				},
				"foo": {
					S: aws.String("baz"),
				},
			},
		},
	}

	s.executor.ExpectExecution("QueryRequest", input, output, nil)

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
	_, err := s.repo.Query(context.Background(), qb, &result)

	s.NoError(err)
	s.Len(result, 2)
	s.EqualValues(expected, result)

	s.executor.AssertExpectations(s.T())
}

func (s *RepositoryTestSuite) TestQuery_Canceled() {
	awsErr := awserr.New(request.CanceledErrorCode, "got canceled", nil)

	input := &dynamodb.QueryInput{
		TableName:              aws.String("applike-test-gosoline-ddb-myModel"),
		KeyConditionExpression: aws.String("#0 = :0"),
		ExpressionAttributeNames: map[string]*string{
			"#0": aws.String("id"),
		},
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":0": {
				N: aws.String("1"),
			},
		},
	}
	s.executor.ExpectExecution("QueryRequest", input, nil, awsErr)

	result := make([]model, 0)

	qb := s.repo.QueryBuilder().WithHash(1)
	_, err := s.repo.Query(context.Background(), qb, &result)

	s.Error(err)

	isRequestCanceled := errors.Is(err, exec.RequestCanceledError)
	s.True(isRequestCanceled)

	s.executor.AssertExpectations(s.T())
}

func (s *RepositoryTestSuite) TestBatchGetItems() {
	input := &dynamodb.BatchGetItemInput{
		RequestItems: map[string]*dynamodb.KeysAndAttributes{
			"applike-test-gosoline-ddb-myModel": {
				Keys: []map[string]*dynamodb.AttributeValue{
					{
						"id":  {N: aws.String("1")},
						"rev": {S: aws.String("0")},
					},
					{
						"id":  {N: aws.String("2")},
						"rev": {S: aws.String("0")},
					},
				},
			},
		},
	}
	output := &dynamodb.BatchGetItemOutput{
		Responses: map[string][]map[string]*dynamodb.AttributeValue{
			"applike-test-gosoline-ddb-myModel": {
				{
					"id":  {N: aws.String("1")},
					"rev": {S: aws.String("0")},
					"foo": {S: aws.String("foo")},
				},
				{
					"id":  {N: aws.String("2")},
					"rev": {S: aws.String("0")},
					"foo": {S: aws.String("bar")},
				},
			},
		},
		UnprocessedKeys: map[string]*dynamodb.KeysAndAttributes{},
	}

	s.executor.ExpectExecution("BatchGetItemRequest", input, output, nil)

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
	_, err := s.repo.BatchGetItems(context.Background(), qb, &result)

	s.NoError(err)
	s.Equal(expected, result)

	s.executor.AssertExpectations(s.T())
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
		RequestItems: map[string][]*dynamodb.WriteRequest{
			"applike-test-gosoline-ddb-myModel": {
				{
					PutRequest: &dynamodb.PutRequest{
						Item: map[string]*dynamodb.AttributeValue{
							"id":  {N: aws.String("1")},
							"rev": {S: aws.String("0")},
							"foo": {S: aws.String("foo")},
						},
					},
				},
				{
					PutRequest: &dynamodb.PutRequest{
						Item: map[string]*dynamodb.AttributeValue{
							"id":  {N: aws.String("2")},
							"rev": {S: aws.String("0")},
							"foo": {S: aws.String("bar")},
						},
					},
				},
			},
		},
	}

	output := &dynamodb.BatchWriteItemOutput{
		UnprocessedItems: map[string][]*dynamodb.WriteRequest{},
	}

	s.executor.ExpectExecution("BatchWriteItemRequest", input, output, nil)

	_, err := s.repo.BatchPutItems(context.Background(), items)

	s.NoError(err)
	s.executor.AssertExpectations(s.T())
}

func (s *RepositoryTestSuite) TestBatchWriteItem_Retry() {
	makeItem := func(id int) model {
		return model{
			Id:  id,
			Rev: fmt.Sprintf("rev %d", id),
			Foo: "data",
		}
	}
	makePutRequest := func(id int) *dynamodb.PutRequest {
		return &dynamodb.PutRequest{
			Item: map[string]*dynamodb.AttributeValue{
				"id":  {N: aws.String(fmt.Sprintf("%d", id))},
				"rev": {S: aws.String(fmt.Sprintf("rev %d", id))},
				"foo": {S: aws.String("data")},
			},
		}
	}

	totalItems := 20
	firstBatchItems := 10

	items := make([]model, 0, totalItems)
	firstInputData := make([]*dynamodb.WriteRequest, 0, totalItems)
	firstOutputData := make([]*dynamodb.WriteRequest, 0, firstBatchItems)
	secondInputData := make([]*dynamodb.WriteRequest, 0, firstBatchItems)
	for i := 0; i < totalItems; i++ {
		items = append(items, makeItem(i))
		firstInputData = append(firstInputData, &dynamodb.WriteRequest{
			PutRequest: makePutRequest(i),
		})
		if i < firstBatchItems {
			secondInputData = append(secondInputData, &dynamodb.WriteRequest{
				PutRequest: makePutRequest(i),
			})
			firstOutputData = append(firstOutputData, &dynamodb.WriteRequest{
				PutRequest: makePutRequest(i),
			})
		}
	}

	firstInput := &dynamodb.BatchWriteItemInput{
		RequestItems: map[string][]*dynamodb.WriteRequest{
			"applike-test-gosoline-ddb-myModel": firstInputData,
		},
	}
	secondInput := &dynamodb.BatchWriteItemInput{
		RequestItems: map[string][]*dynamodb.WriteRequest{
			"applike-test-gosoline-ddb-myModel": secondInputData,
		},
	}

	firstOutput := &dynamodb.BatchWriteItemOutput{
		UnprocessedItems: map[string][]*dynamodb.WriteRequest{
			"applike-test-gosoline-ddb-myModel": firstOutputData,
		},
	}
	secondOutput := &dynamodb.BatchWriteItemOutput{
		UnprocessedItems: map[string][]*dynamodb.WriteRequest{},
	}

	s.executor.ExpectExecution("BatchWriteItemRequest", firstInput, firstOutput, nil)
	s.executor.ExpectExecution("BatchWriteItemRequest", secondInput, firstOutput, nil)
	s.executor.ExpectExecution("BatchWriteItemRequest", secondInput, secondOutput, nil)

	_, err := s.repo.BatchPutItems(context.Background(), items)

	s.NoError(err)
	s.executor.AssertExpectations(s.T())
}

func (s *RepositoryTestSuite) TestPutItem() {
	item := model{
		Id:  1,
		Rev: "0",
		Foo: "foo",
	}

	input := &dynamodb.PutItemInput{
		TableName: aws.String("applike-test-gosoline-ddb-myModel"),
		Item: map[string]*dynamodb.AttributeValue{
			"id": {
				N: aws.String("1"),
			},
			"rev": {
				S: aws.String("0"),
			},
			"foo": {
				S: aws.String("foo"),
			},
		},
	}
	output := &dynamodb.PutItemOutput{}

	s.executor.ExpectExecution("PutItemRequest", input, output, nil)

	res, err := s.repo.PutItem(context.Background(), nil, item)

	s.NoError(err)
	s.False(res.ConditionalCheckFailed)
	s.executor.AssertExpectations(s.T())
}

func (s *RepositoryTestSuite) TestUpdate() {
	input := &dynamodb.UpdateItemInput{
		TableName: aws.String("applike-test-gosoline-ddb-myModel"),
		Key: map[string]*dynamodb.AttributeValue{
			"id": {
				N: aws.String("1"),
			},
			"rev": {
				S: aws.String("0"),
			},
		},
		ExpressionAttributeNames: map[string]*string{
			"#0": aws.String("foo"),
		},
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":0": {
				S: aws.String("bar"),
			},
		},
		UpdateExpression: aws.String("SET #0 = :0\n"),
		ReturnValues:     aws.String(dynamodb.ReturnValueAllNew),
	}
	output := &dynamodb.UpdateItemOutput{
		Attributes: map[string]*dynamodb.AttributeValue{
			"id": {
				N: aws.String("1"),
			},
			"rev": {
				S: aws.String("0"),
			},
			"foo": {
				S: aws.String("bar"),
			},
		},
	}

	s.executor.ExpectExecution("UpdateItemRequest", input, output, nil)

	updatedItem := &model{
		Id:  1,
		Rev: "0",
	}
	ub := s.repo.UpdateItemBuilder().Set("foo", "bar").ReturnAllNew()
	res, err := s.repo.UpdateItem(context.Background(), ub, updatedItem)

	expectedItem := &model{
		Id:  1,
		Rev: "0",
		Foo: "bar",
	}

	s.NoError(err)
	s.False(res.ConditionalCheckFailed)
	s.EqualValues(expectedItem, updatedItem)
	s.executor.AssertExpectations(s.T())
}

func (s *RepositoryTestSuite) TestDeleteItem() {
	input := &dynamodb.DeleteItemInput{
		ConditionExpression: aws.String("#0 = :0"),
		ExpressionAttributeNames: map[string]*string{
			"#0": aws.String("foo"),
		},
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":0": {
				S: aws.String("bar"),
			},
		},
		Key: map[string]*dynamodb.AttributeValue{
			"id": {
				N: aws.String("1"),
			},
			"rev": {
				S: aws.String("0"),
			},
		},
		ReturnValues: aws.String(dynamodb.ReturnValueAllOld),
		TableName:    aws.String("applike-test-gosoline-ddb-myModel"),
	}
	output := &dynamodb.DeleteItemOutput{
		Attributes: map[string]*dynamodb.AttributeValue{
			"id": {
				N: aws.String("1"),
			},
			"rev": {
				S: aws.String("0"),
			},
			"foo": {
				S: aws.String("bar"),
			},
		},
	}

	s.executor.ExpectExecution("DeleteItemRequest", input, output, nil)

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
	res, err := s.repo.DeleteItem(context.Background(), db, &item)

	s.NoError(err)
	s.False(res.ConditionalCheckFailed)
	s.Equal(expected, item)
	s.executor.AssertExpectations(s.T())
}

func TestRepositoryTestSuite(t *testing.T) {
	suite.Run(t, new(RepositoryTestSuite))
}
