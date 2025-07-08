package ddb

import (
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/expression"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

//go:generate go run github.com/vektra/mockery/v2 --name PutItemBuilder
type PutItemBuilder interface {
	WithCondition(cond expression.ConditionBuilder) PutItemBuilder
	ReturnNone() PutItemBuilder
	ReturnAllOld() PutItemBuilder
	Build(item interface{}) (*dynamodb.PutItemInput, error)
}

type putItemBuilder struct {
	metadata   *Metadata
	condition  *expression.ConditionBuilder
	returnType types.ReturnValue
}

func NewPutItemBuilder(metadata *Metadata) PutItemBuilder {
	return &putItemBuilder{
		metadata: metadata,
	}
}

func (b *putItemBuilder) WithCondition(cond expression.ConditionBuilder) PutItemBuilder {
	b.condition = &cond

	return b
}

func (b *putItemBuilder) ReturnNone() PutItemBuilder {
	b.returnType = types.ReturnValueNone

	return b
}

func (b *putItemBuilder) ReturnAllOld() PutItemBuilder {
	b.returnType = types.ReturnValueAllOld

	return b
}

func (b *putItemBuilder) Build(item interface{}) (*dynamodb.PutItemInput, error) {
	if b.returnType != "" && b.returnType != types.ReturnValueNone && !isPointer(item) {
		return nil, fmt.Errorf("the provided old value has to be a pointer")
	}

	var err error
	expr := expression.Expression{}

	if b.condition != nil {
		expr, err = expression.NewBuilder().WithCondition(*b.condition).Build()
	}

	if err != nil {
		return nil, fmt.Errorf("could not build condition: %w", err)
	}

	input := &dynamodb.PutItemInput{
		TableName:                 aws.String(b.metadata.TableName),
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
		ConditionExpression:       expr.Condition(),
		ReturnConsumedCapacity:    types.ReturnConsumedCapacityIndexes,
		ReturnValues:              b.returnType,
	}

	marshalled, err := MarshalMap(item)
	if err != nil {
		return nil, err
	}

	input.Item = marshalled

	return input, err
}
