package ddb

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/expression"
)

//go:generate mockery -name DeleteItemBuilder
type DeleteItemBuilder interface {
	WithHash(hashValue interface{}) DeleteItemBuilder
	WithRange(rangeValue interface{}) DeleteItemBuilder
	WithCondition(cond expression.ConditionBuilder) DeleteItemBuilder
	ReturnNone() DeleteItemBuilder
	ReturnAllOld() DeleteItemBuilder
	Build(item interface{}) (*dynamodb.DeleteItemInput, error)
}

type deleteItemBuilder struct {
	err        error
	metadata   *Metadata
	keyBuilder keyBuilder
	condition  *expression.ConditionBuilder
	returnType *string
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
	b.returnType = aws.String(dynamodb.ReturnValueNone)

	return b
}

func (b *deleteItemBuilder) ReturnAllOld() DeleteItemBuilder {
	b.returnType = aws.String(dynamodb.ReturnValueAllOld)

	return b
}

func (b *deleteItemBuilder) Build(item interface{}) (*dynamodb.DeleteItemInput, error) {
	if b.returnType != nil && b.returnType != aws.String(dynamodb.ReturnValueAllOld) && !isPointer(item) {
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
		ReturnValues:              b.returnType,
	}

	return input, err
}
