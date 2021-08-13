package ddb

import (
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

type TransactDeleteItem struct {
	builder DeleteItemBuilder
	item    interface{}
}

func (b *TransactDeleteItem) Build() (*types.TransactWriteItem, error) {
	if !isPointer(b.item) {
		return nil, fmt.Errorf("item must be a pointer")
	}

	entry, err := b.builder.Build(b.item)
	if err != nil {
		return nil, fmt.Errorf("could not built entry for transact delete item: %w", err)
	}

	item := &types.TransactWriteItem{
		Delete: &types.Delete{
			ConditionExpression:                 entry.ConditionExpression,
			ExpressionAttributeNames:            entry.ExpressionAttributeNames,
			ExpressionAttributeValues:           entry.ExpressionAttributeValues,
			Key:                                 entry.Key,
			ReturnValuesOnConditionCheckFailure: types.ReturnValuesOnConditionCheckFailure(entry.ReturnValues),
			TableName:                           entry.TableName,
		},
	}

	return item, nil
}

func (b *TransactDeleteItem) GetItem() interface{} {
	return b.item
}
