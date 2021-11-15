package ddb

import (
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

type TransactDeleteItem struct {
	Builder DeleteItemBuilder
	Item    interface{}
}

func (b *TransactDeleteItem) Build() (*types.TransactWriteItem, error) {
	if !isPointer(b.Item) {
		return nil, fmt.Errorf("item must be a pointer")
	}

	entry, err := b.Builder.Build(b.Item)
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
	return b.Item
}
