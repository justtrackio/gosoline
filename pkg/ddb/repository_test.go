package ddb_test

import (
	"context"
	"fmt"
	cloudMocks "github.com/applike/gosoline/pkg/cloud/mocks"
	"github.com/applike/gosoline/pkg/ddb"
	"github.com/applike/gosoline/pkg/mdl"
	monMocks "github.com/applike/gosoline/pkg/mon/mocks"
	"github.com/applike/gosoline/pkg/tracing"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
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

func TestRepository_GetItem(t *testing.T) {
	client, repo := getMocks()

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

	client.On("GetItemWithContext", mock.AnythingOfType("*context.emptyCtx"), input).Return(output, nil)

	qb := repo.GetItemBuilder().WithHash(1).WithRange("0")
	res, err := repo.GetItem(context.Background(), qb, &item)

	expected := model{
		Id:  1,
		Rev: "0",
		Foo: "bar",
	}

	assert.NoError(t, err)
	assert.True(t, res.IsFound)
	assert.EqualValues(t, expected, item)

	client.AssertExpectations(t)
}

func TestRepository_GetItem_FromItem(t *testing.T) {
	client, repo := getMocks()

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
	client.On("GetItemWithContext", mock.AnythingOfType("*context.emptyCtx"), input).Return(output, nil)

	item := model{
		Id:  5,
		Rev: "abc",
	}

	qb := repo.GetItemBuilder().WithHash(5).WithRange("abc")
	res, err := repo.GetItem(context.Background(), qb, &item)

	expected := model{
		Id:  5,
		Rev: "abc",
		Foo: "baz",
	}

	assert.NoError(t, err)
	assert.True(t, res.IsFound)
	assert.EqualValues(t, expected, item)
}

func TestRepository_GetItemNotFound(t *testing.T) {
	client, repo := getMocks()
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
	client.On("GetItemWithContext", mock.AnythingOfType("*context.emptyCtx"), input).Return(output, nil)

	qb := repo.GetItemBuilder().WithHash(1).WithRange("0")
	res, err := repo.GetItem(context.Background(), qb, &item)

	assert.NoError(t, err)
	assert.False(t, res.IsFound)

	client.AssertExpectations(t)
}

func TestRepository_GetItemProjection(t *testing.T) {
	client, repo := getMocks()

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
	client.On("GetItemWithContext", mock.AnythingOfType("*context.emptyCtx"), input).Return(output, nil)

	item := projection{}

	qb := repo.GetItemBuilder().WithHash(1).WithRange("0").WithProjection(item)
	res, err := repo.GetItem(context.Background(), qb, &item)

	expected := projection{
		Id: 1,
	}

	assert.NoError(t, err)
	assert.True(t, res.IsFound)
	assert.EqualValues(t, expected, item)

	client.AssertExpectations(t)
}

func TestRepository_Query(t *testing.T) {
	client, repo := getMocks()

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

	client.On("QueryWithContext", mock.AnythingOfType("*context.emptyCtx"), input).Return(output, nil)

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

	qb := repo.QueryBuilder().WithHash(1)
	_, err := repo.Query(context.Background(), qb, &result)

	assert.NoError(t, err)
	assert.Len(t, result, 2)
	assert.EqualValues(t, expected, result)

	client.AssertExpectations(t)
}

func TestRepository_BatchGetItems(t *testing.T) {
	client, repo := getMocks()

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
	client.On("BatchGetItemWithContext", mock.AnythingOfType("*context.emptyCtx"), input).Return(output, nil)

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

	qb := repo.BatchGetItemsBuilder().WithKeys(1, "0").WithKeys(2, "0")
	_, err := repo.BatchGetItems(context.Background(), qb, &result)

	assert.NoError(t, err)
	assert.Equal(t, expected, result)

	client.AssertExpectations(t)
}

func TestRepository_BatchWriteItem(t *testing.T) {
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

	client, repo := getMocks()

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

	client.On("BatchWriteItemWithContext", mock.AnythingOfType("*context.emptyCtx"), input).Return(output, nil)

	_, err := repo.BatchPutItems(context.Background(), items)

	assert.NoError(t, err)
	client.AssertExpectations(t)
}

func TestRepository_BatchWriteItem_Retry(t *testing.T) {
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

	client, repo := getMocks()

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

	client.On("BatchWriteItemWithContext", mock.AnythingOfType("*context.emptyCtx"), firstInput).Return(firstOutput, nil).Once()
	client.On("BatchWriteItemWithContext", mock.AnythingOfType("*context.emptyCtx"), secondInput).Return(firstOutput, nil).Once()
	client.On("BatchWriteItemWithContext", mock.AnythingOfType("*context.emptyCtx"), secondInput).Return(secondOutput, nil).Once()

	_, err := repo.BatchPutItems(context.Background(), items)

	assert.NoError(t, err)
	client.AssertExpectations(t)
}

func TestRepository_PutItem(t *testing.T) {
	client, r := getMocks()

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
	client.On("PutItemWithContext", mock.AnythingOfType("*context.emptyCtx"), input).Return(output, nil)

	res, err := r.PutItem(context.Background(), nil, item)

	assert.NoError(t, err)
	assert.False(t, res.ConditionalCheckFailed)
	client.AssertExpectations(t)
}

func TestRepository_Update(t *testing.T) {
	client, repo := getMocks()

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
	client.On("UpdateItemWithContext", mock.AnythingOfType("*context.emptyCtx"), input).Return(output, nil)

	updatedItem := &model{
		Id:  1,
		Rev: "0",
	}
	ub := repo.UpdateItemBuilder().Set("foo", "bar").ReturnAllNew()
	res, err := repo.UpdateItem(context.Background(), ub, updatedItem)

	expectedItem := &model{
		Id:  1,
		Rev: "0",
		Foo: "bar",
	}

	assert.NoError(t, err)
	assert.False(t, res.ConditionalCheckFailed)
	assert.EqualValues(t, expectedItem, updatedItem)
	client.AssertExpectations(t)
}

func TestRepository_DeleteItem(t *testing.T) {
	client, repo := getMocks()

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

	client.On("DeleteItemWithContext", mock.AnythingOfType("*context.emptyCtx"), input).Return(output, nil)

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

	db := repo.DeleteItemBuilder().WithCondition(ddb.Eq("foo", "bar")).ReturnAllOld()
	res, err := repo.DeleteItem(context.Background(), db, &item)

	assert.NoError(t, err)
	assert.False(t, res.ConditionalCheckFailed)
	assert.Equal(t, expected, item)
	client.AssertExpectations(t)
}

func getMocks() (*cloudMocks.DynamoDBAPI, ddb.Repository) {
	logger := monMocks.NewLoggerMockedAll()
	tracer := tracing.NewNoopTracer()
	client := new(cloudMocks.DynamoDBAPI)

	repo := ddb.NewWithInterfaces(logger, tracer, client, &ddb.Settings{
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

	return client, repo
}
