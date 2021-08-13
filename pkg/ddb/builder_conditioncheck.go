package ddb

import (
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/expression"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/hashicorp/go-multierror"
)

//go:generate mockery --name ConditionCheckBuilder
type ConditionCheckBuilder interface {
	Build(result interface{}) (*types.ConditionCheck, error)
	ReturnNone() ConditionCheckBuilder
	ReturnAllOld() ConditionCheckBuilder
	WithHash(hashValue interface{}) ConditionCheckBuilder
	WithRange(rangeValue interface{}) ConditionCheckBuilder
	WithKeys(keys ...interface{}) ConditionCheckBuilder
	WithCondition(cond expression.ConditionBuilder) ConditionCheckBuilder
}

type conditionCheckBuilder struct {
	condition  *expression.ConditionBuilder
	err        error
	metadata   *Metadata
	keyBuilder keyBuilder
	returnType types.ReturnValuesOnConditionCheckFailure
}

func NewConditionCheckBuilder(metadata *Metadata) ConditionCheckBuilder {
	return &conditionCheckBuilder{
		metadata: metadata,
		keyBuilder: keyBuilder{
			metadata: metadata.Main,
		},
	}
}

func (b *conditionCheckBuilder) ReturnNone() ConditionCheckBuilder {
	b.returnType = types.ReturnValuesOnConditionCheckFailureNone
	return b
}

func (b *conditionCheckBuilder) ReturnAllOld() ConditionCheckBuilder {
	b.returnType = types.ReturnValuesOnConditionCheckFailureAllOld
	return b
}

func (b *conditionCheckBuilder) WithCondition(cond expression.ConditionBuilder) ConditionCheckBuilder {
	b.condition = &cond
	return b
}

func (b *conditionCheckBuilder) WithHash(hashValue interface{}) ConditionCheckBuilder {
	b.keyBuilder.withHash(hashValue)
	return b
}

func (b *conditionCheckBuilder) WithRange(rangeValue interface{}) ConditionCheckBuilder {
	b.keyBuilder.withRange(rangeValue)
	return b
}

func (b *conditionCheckBuilder) WithKeys(keys ...interface{}) ConditionCheckBuilder {
	if len(keys) == 0 {
		return b
	}

	b.WithHash(keys[0])

	if len(keys) > 2 {
		b.err = multierror.Append(b.err, fmt.Errorf("more than two keys provided for WithKeys"))
		return b
	}

	b.WithRange(keys[1])

	return b
}

func (b *conditionCheckBuilder) Build(result interface{}) (*types.ConditionCheck, error) {
	if b.err != nil {
		return nil, b.err
	}

	keys, err := b.keyBuilder.buildKey(result)
	if err != nil {
		return nil, err
	}

	expr, err := b.buildExpression()
	if err != nil {
		return nil, err
	}

	input := &types.ConditionCheck{
		ConditionExpression:                 expr.Condition(),
		ExpressionAttributeNames:            expr.Names(),
		ExpressionAttributeValues:           expr.Values(),
		Key:                                 keys,
		ReturnValuesOnConditionCheckFailure: b.returnType,
		TableName:                           aws.String(b.metadata.TableName),
	}

	return input, nil
}

func (b *conditionCheckBuilder) buildExpression() (expression.Expression, error) {
	if b.condition == nil {
		return expression.Expression{}, nil
	}

	expr, err := expression.
		NewBuilder().
		WithCondition(*b.condition).
		Build()

	return expr, err
}
