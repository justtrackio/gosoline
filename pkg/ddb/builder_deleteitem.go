package ddb

import (
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/expression"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

//go:generate go run github.com/vektra/mockery/v2 --name DeleteItemBuilder
type DeleteItemBuilder interface {
	WithHash(hashValue interface{}) DeleteItemBuilder
	WithRange(rangeValue interface{}) DeleteItemBuilder
	WithCondition(cond expression.ConditionBuilder) DeleteItemBuilder
	ReturnNone() DeleteItemBuilder
	ReturnAllOld() DeleteItemBuilder
	Build(item interface{}) (*dynamodb.DeleteItemInput, error)
}

type deleteItemBuilder struct {
	metadata   *Metadata
	keyBuilder keyBuilder
	condition  *expression.ConditionBuilder
	returnType types.ReturnValue
}

func NewDeleteItemBuilder(metadata *Metadata) DeleteItemBuilder {
	return &deleteItemBuilder{
		metadata: metadata,
		keyBuilder: keyBuilder{
			metadata: metadata.Main,
		},
	}
}

func (b *deleteItemBuilder) WithHash(hashValue interface{}) DeleteItemBuilder {
	b.keyBuilder.withHash(hashValue)

	return b
}

func (b *deleteItemBuilder) WithRange(rangeValue interface{}) DeleteItemBuilder {
	b.keyBuilder.withRange(rangeValue)

	return b
}

func (b *deleteItemBuilder) WithCondition(cond expression.ConditionBuilder) DeleteItemBuilder {
	b.condition = &cond

	return b
}

func (b *deleteItemBuilder) ReturnNone() DeleteItemBuilder {
	b.returnType = types.ReturnValueNone

	return b
}

func (b *deleteItemBuilder) ReturnAllOld() DeleteItemBuilder {
	b.returnType = types.ReturnValueAllOld

	return b
}

func (b *deleteItemBuilder) Build(item interface{}) (*dynamodb.DeleteItemInput, error) {
	if b.returnType != "" && b.returnType != types.ReturnValueNone && !isPointer(item) {
		return nil, fmt.Errorf("the provided old value has to be a pointer")
	}

	var err error
	expr := expression.Expression{}

	if b.condition != nil {
		expr, err = expression.NewBuilder().WithCondition(*b.condition).Build()
	}

	if err != nil {
		return nil, err
	}

	key, err := b.keyBuilder.buildKey(item)
	if err != nil {
		return nil, err
	}

	input := &dynamodb.DeleteItemInput{
		TableName:                 aws.String(b.metadata.TableName),
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
		ConditionExpression:       expr.Condition(),
		Key:                       key,
		ReturnConsumedCapacity:    types.ReturnConsumedCapacityIndexes,
		ReturnValues:              b.returnType,
	}

	return input, err
}
