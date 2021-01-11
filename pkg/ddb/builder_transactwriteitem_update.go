package ddb

import (
	"fmt"
	"github.com/aws/aws-sdk-go/service/dynamodb"
)

type TransactUpdateItem struct {
	Builder UpdateItemBuilder
	Item    interface{}
}

func (b *TransactUpdateItem) Build() (*dynamodb.TransactWriteItem, error) {
	if !isPointer(b.Item) {
		return nil, fmt.Errorf("item must be a pointer")
	}

	entry, err := b.Builder.Build(b.Item)
	if err != nil {
		return nil, fmt.Errorf("could not built entry for transact update item: %w", err)
	}

	item := &dynamodb.TransactWriteItem{
		Update: &dynamodb.Update{
			ConditionExpression:                 entry.ConditionExpression,
			ExpressionAttributeNames:            entry.ExpressionAttributeNames,
			ExpressionAttributeValues:           entry.ExpressionAttributeValues,
			Key:                                 entry.Key,
			TableName:                           entry.TableName,
			UpdateExpression:                    entry.UpdateExpression,
			ReturnValuesOnConditionCheckFailure: entry.ReturnValues,
		},
	}

	return item, nil
}

func (b *TransactUpdateItem) GetItem() interface{} {
	return b.Item
}
