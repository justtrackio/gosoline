package ddb

import (
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/expression"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

//go:generate go run github.com/vektra/mockery/v2 --name UpdateItemBuilder
type UpdateItemBuilder interface {
	WithHash(hashValue any) UpdateItemBuilder
	WithRange(rangeValue any) UpdateItemBuilder
	WithCondition(cond expression.ConditionBuilder) UpdateItemBuilder
	Add(path string, value any) UpdateItemBuilder
	Delete(path string, value any) UpdateItemBuilder
	Set(path string, value any) UpdateItemBuilder
	SetMap(values map[string]any) UpdateItemBuilder
	SetIfNotExist(path string, value any) UpdateItemBuilder
	Remove(path string) UpdateItemBuilder
	RemoveMultiple(paths ...string) UpdateItemBuilder
	ReturnNone() UpdateItemBuilder
	ReturnAllOld() UpdateItemBuilder
	ReturnUpdatedOld() UpdateItemBuilder
	ReturnAllNew() UpdateItemBuilder
	ReturnUpdatedNew() UpdateItemBuilder
	Build(item any) (*dynamodb.UpdateItemInput, error)
}

type updateItemBuilder struct {
	metadata      *Metadata
	keyBuilder    keyBuilder
	condition     *expression.ConditionBuilder
	updateBuilder *expression.UpdateBuilder
	returnType    types.ReturnValue
}

func NewUpdateItemBuilder(metadata *Metadata) UpdateItemBuilder {
	return &updateItemBuilder{
		metadata: metadata,
		keyBuilder: keyBuilder{
			metadata: metadata.Main,
		},
	}
}

func (b *updateItemBuilder) WithHash(hashValue any) UpdateItemBuilder {
	b.keyBuilder.withHash(hashValue)

	return b
}

func (b *updateItemBuilder) WithRange(rangeValue any) UpdateItemBuilder {
	b.keyBuilder.withRange(rangeValue)

	return b
}

func (b *updateItemBuilder) WithCondition(cond expression.ConditionBuilder) UpdateItemBuilder {
	b.condition = &cond

	return b
}

func (b *updateItemBuilder) Add(path string, value any) UpdateItemBuilder {
	return b.update(func() expression.UpdateBuilder {
		return b.updateBuilder.Add(expression.Name(path), expression.Value(value))
	})
}

func (b *updateItemBuilder) Delete(path string, value any) UpdateItemBuilder {
	return b.update(func() expression.UpdateBuilder {
		return b.updateBuilder.Delete(expression.Name(path), expression.Value(value))
	})
}

func (b *updateItemBuilder) Set(path string, value any) UpdateItemBuilder {
	return b.update(func() expression.UpdateBuilder {
		return b.updateBuilder.Set(expression.Name(path), expression.Value(value))
	})
}

func (b *updateItemBuilder) SetMap(values map[string]any) UpdateItemBuilder {
	for k, v := range values {
		b.Set(k, v)
	}

	return b
}

func (b *updateItemBuilder) SetIfNotExist(path string, value any) UpdateItemBuilder {
	return b.update(func() expression.UpdateBuilder {
		return b.updateBuilder.Set(expression.Name(path), expression.IfNotExists(expression.Name(path), expression.Value(value)))
	})
}

func (b *updateItemBuilder) Remove(path string) UpdateItemBuilder {
	return b.update(func() expression.UpdateBuilder {
		return b.updateBuilder.Remove(expression.Name(path))
	})
}

func (b *updateItemBuilder) RemoveMultiple(paths ...string) UpdateItemBuilder {
	for _, p := range paths {
		b.Remove(p)
	}

	return b
}

func (b *updateItemBuilder) ReturnNone() UpdateItemBuilder {
	b.returnType = types.ReturnValueNone

	return b
}

func (b *updateItemBuilder) ReturnAllOld() UpdateItemBuilder {
	b.returnType = types.ReturnValueAllOld

	return b
}

func (b *updateItemBuilder) ReturnUpdatedOld() UpdateItemBuilder {
	b.returnType = types.ReturnValueUpdatedOld

	return b
}

func (b *updateItemBuilder) ReturnAllNew() UpdateItemBuilder {
	b.returnType = types.ReturnValueAllNew

	return b
}

func (b *updateItemBuilder) ReturnUpdatedNew() UpdateItemBuilder {
	b.returnType = types.ReturnValueUpdatedNew

	return b
}

func (b *updateItemBuilder) Build(item any) (*dynamodb.UpdateItemInput, error) {
	keys, err := b.keyBuilder.buildKey(item)
	if err != nil {
		return nil, err
	}

	if b.returnType != "" && b.returnType != types.ReturnValueNone && !isPointer(item) {
		return nil, fmt.Errorf("value for returning the updated item is not a pointer")
	}

	expr, err := b.buildExpression()
	if err != nil {
		return nil, err
	}

	input := &dynamodb.UpdateItemInput{
		TableName:                 aws.String(b.metadata.TableName),
		Key:                       keys,
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
		ConditionExpression:       expr.Condition(),
		UpdateExpression:          expr.Update(),
		ReturnConsumedCapacity:    types.ReturnConsumedCapacityIndexes,
		ReturnValues:              b.returnType,
	}

	return input, err
}

func (b *updateItemBuilder) buildExpression() (expression.Expression, error) {
	if b.updateBuilder == nil && b.condition == nil {
		return expression.Expression{}, nil
	}

	exprBuilder := expression.NewBuilder()

	if b.updateBuilder != nil {
		exprBuilder = exprBuilder.WithUpdate(*b.updateBuilder)
	}

	if b.condition != nil {
		exprBuilder = exprBuilder.WithCondition(*b.condition)
	}

	return exprBuilder.Build()
}

func (b *updateItemBuilder) update(callback func() expression.UpdateBuilder) *updateItemBuilder {
	if b.updateBuilder == nil {
		ub := expression.UpdateBuilder{}
		b.updateBuilder = &ub
	}

	ub := callback()
	b.updateBuilder = &ub

	return b
}
