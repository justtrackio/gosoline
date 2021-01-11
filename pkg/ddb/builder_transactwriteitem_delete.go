package ddb

import (
	"fmt"
	"github.com/aws/aws-sdk-go/service/dynamodb"
)

type TransactDeleteItem struct {
	builder DeleteItemBuilder
	item    interface{}
}

func (b *TransactDeleteItem) Build() (*dynamodb.TransactWriteItem, error) {
	if !isPointer(b.item) {
		return nil, fmt.Errorf("item must be a pointer")
	}

	entry, err := b.builder.Build(b.item)
	if err != nil {
		return nil, fmt.Errorf("could not built entry for transact delete item: %w", err)
	}

	item := &dynamodb.TransactWriteItem{
		Delete: &dynamodb.Delete{
			ConditionExpression:                 entry.ConditionExpression,
			ExpressionAttributeNames:            entry.ExpressionAttributeNames,
			ExpressionAttributeValues:           entry.ExpressionAttributeValues,
			Key:                                 entry.Key,
			ReturnValuesOnConditionCheckFailure: entry.ReturnValues,
			TableName:                           entry.TableName,
		},
	}

	return item, nil
}

func (b *TransactDeleteItem) GetItem() interface{} {
	return b.item
}
