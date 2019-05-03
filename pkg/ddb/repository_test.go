package ddb_test

import (
	"context"
	"errors"
	"github.com/adjoeio/djoemo"
	cloudMocks "github.com/applike/gosoline/pkg/cloud/mocks"
	"github.com/applike/gosoline/pkg/ddb"
	"github.com/applike/gosoline/pkg/ddb/mocks"
	"github.com/applike/gosoline/pkg/mdl"
	monMocks "github.com/applike/gosoline/pkg/mon/mocks"
	"github.com/applike/gosoline/pkg/tracing"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"testing"
)

type model struct {
	Id  int    `dynamo:"id,hash"`
	Foo string `dynamo:"foo"`
}

func TestRepository_CreateTable(t *testing.T) {
	dyn, r, _ := getMocks()

	input := &dynamodb.DescribeTableInput{
		TableName: aws.String("test-test-test-test-test"),
	}
	dyn.On("DescribeTable", input).Return(nil, errors.New(dynamodb.ErrCodeResourceNotFoundException))

	bla := &dynamodb.CreateTableInput{
		AttributeDefinitions: []*dynamodb.AttributeDefinition{{
			AttributeName: aws.String("id"),
			AttributeType: aws.String(dynamodb.ScalarAttributeTypeN),
		}},
		KeySchema: []*dynamodb.KeySchemaElement{{
			AttributeName: aws.String("id"),
			KeyType:       aws.String(dynamodb.KeyTypeHash),
		}},
		ProvisionedThroughput: &dynamodb.ProvisionedThroughput{
			ReadCapacityUnits:  aws.Int64(1),
			WriteCapacityUnits: aws.Int64(2),
		},
		TableName: aws.String("test-test-test-test-test"),
	}
	dyn.On("CreateTableWithContext", mock.Anything, bla).Return(nil, nil)

	err := r.CreateTable(model{})

	dyn.AssertExpectations(t)
	assert.Nil(t, err, "there should be no error")
}

func TestRepository_GetItem(t *testing.T) {
	dyn, r, dj := getMocks()

	var item interface{}
	qb := ddb.NewQueryBuilder("test-test-test-test-test")

	dj.On("GetItemWithContext", mock.AnythingOfType("*context.emptyCtx"), qb.Build(), &item).Return(true, nil)

	ok, err := r.GetItem(context.TODO(), qb, &item)

	assert.NoError(t, err)
	assert.True(t, ok)

	dyn.AssertExpectations(t)
	dj.AssertExpectations(t)
}

func TestRepository_GetItemNotFound(t *testing.T) {
	dyn, r, dj := getMocks()
	var item interface{}
	qb := ddb.NewQueryBuilder("test-test-test-test-test")

	dj.On("GetItemWithContext", mock.AnythingOfType("*context.emptyCtx"), qb.Build(), &item).Return(false, nil)

	ok, err := r.GetItem(context.TODO(), qb, &item)

	assert.NoError(t, err)
	assert.False(t, ok)

	dyn.AssertExpectations(t)
	dj.AssertExpectations(t)
}

func TestRepository_GetItems(t *testing.T) {
	dyn, r, dj := getMocks()

	var item []interface{}
	qb := ddb.NewQueryBuilder("test-test-test-test-test")

	dj.On("GetItemsWithContext", mock.AnythingOfType("*context.emptyCtx"), qb.Build(), &item).Return(true, nil)

	found, err := r.GetItems(context.TODO(), qb, &item)

	assert.NoError(t, err)
	assert.False(t, found)

	dyn.AssertExpectations(t)
	dj.AssertExpectations(t)
}

func TestRepository_GetItemsInvalidInput(t *testing.T) {
	dyn, r, dj := getMocks()

	var item []interface{}
	qb := ddb.NewQueryBuilder("test-test-test-test-test")

	dj.On("GetItemsWithContext", mock.AnythingOfType("*context.emptyCtx"), qb.Build(), item).Return(true, nil)

	found, err := r.GetItems(context.TODO(), qb, item)

	assert.Error(t, err)
	assert.False(t, found)

	dyn.AssertExpectations(t)
	dj.AssertExpectations(t)
}

func TestRepository_GIndex(t *testing.T) {
	dyn, r, dj := getMocks()
	var item interface{}

	qb := ddb.NewQueryBuilder("test-test-test-test-test")

	idx := "idx"

	qbMock := &mocks.QueryBuilder{}
	qbMock.On("Index").Return(&idx)
	qbMock.On("Build").Return(qb.Build())

	dj.On("GetItemWithContext", mock.AnythingOfType("*context.emptyCtx"), qb.Build(), &item).Return(false, nil)
	dj.On("GIndex", idx).Return(dj)

	ok, err := r.GetItem(context.TODO(), qbMock, &item)

	assert.NoError(t, err)
	assert.False(t, ok)

	dyn.AssertExpectations(t)
	dj.AssertExpectations(t)
}

func TestRepository_Save(t *testing.T) {
	dyn, r, dj := getMocks()

	var item interface{}

	dj.On("SaveItemWithContext", mock.AnythingOfType("*context.emptyCtx"), r.QueryBuilder().WithHash("", "").Build(), item).Return(nil)

	err := r.Save(context.TODO(), item)

	assert.NoError(t, err)

	dyn.AssertExpectations(t)
	dj.AssertExpectations(t)
}

func TestRepository_Update(t *testing.T) {
	dyn, r, dj := getMocks()

	qb := r.QueryBuilder().WithHash("", "")
	dj.On("UpdateWithContext", mock.AnythingOfType("*context.emptyCtx"), mock.AnythingOfType("djoemo.UpdateExpression"), qb.Build(), mock.Anything).Run(func(args mock.Arguments) {
		assert.IsType(t, map[string]interface{}{}, args.Get(3))
	}).Return(nil)

	err := r.Update(context.TODO(), djoemo.SetExpr, qb, map[string]interface{}{
		"'value' = 'value' + ?": []interface{}{42},
	})
	assert.NoError(t, err)

	err = r.Update(context.TODO(), djoemo.Set, qb, map[string]interface{}{
		"value": 42,
	})
	assert.NoError(t, err)
	dyn.AssertExpectations(t)
	dj.AssertExpectations(t)
}

func getMocks() (*cloudMocks.DynamoDBAPI, ddb.Repository, *mocks.DjoemoeRepository) {
	logger := monMocks.NewLoggerMockedAll()
	tracer := tracing.NewNoopTracer()
	dyn := new(cloudMocks.DynamoDBAPI)
	dj := new(mocks.DjoemoeRepository)

	r := ddb.NewFromInterfaces(logger, tracer, dyn, ddb.Settings{
		ModelId: mdl.ModelId{
			Project:     "test",
			Environment: "test",
			Family:      "test",
			Application: "test",
			Name:        "test",
		},
		AutoCreate:         true,
		ReadCapacityUnits:  1,
		WriteCapacityUnits: 2,
	}, dj)

	return dyn, r, dj
}
