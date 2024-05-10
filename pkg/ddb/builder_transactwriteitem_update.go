package ddb

import (
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

type TransactUpdateItem struct {
	Builder UpdateItemBuilder
	Item    any
}

func (b *TransactUpdateItem) Build() (*types.TransactWriteItem, error) {
	if !isPointer(b.Item) {
		return nil, fmt.Errorf("item must be a pointer")
	}

	entry, err := b.Builder.Build(b.Item)
	if err != nil {
		return nil, fmt.Errorf("could not built entry for transact update item: %w", err)
	}

	item := &types.TransactWriteItem{
		Update: &types.Update{
			ConditionExpression:                 entry.ConditionExpression,
			ExpressionAttributeNames:            entry.ExpressionAttributeNames,
			ExpressionAttributeValues:           entry.ExpressionAttributeValues,
			Key:                                 entry.Key,
			TableName:                           entry.TableName,
			UpdateExpression:                    entry.UpdateExpression,
			ReturnValuesOnConditionCheckFailure: types.ReturnValuesOnConditionCheckFailure(entry.ReturnValues),
		},
	}

	return item, nil
}

func (b *TransactUpdateItem) GetItem() any {
	return b.Item
}
