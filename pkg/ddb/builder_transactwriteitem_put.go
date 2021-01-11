package ddb

import (
	"fmt"
	"github.com/aws/aws-sdk-go/service/dynamodb"
)

type TransactPutItem struct {
	Builder PutItemBuilder
	Item    interface{}
}

func NewTransactionPutItemBuilder() *TransactPutItem {
	return &TransactPutItem{}
}

func (b *TransactPutItem) Build() (*dynamodb.TransactWriteItem, error) {
	if !isPointer(b.Item) {
		return nil, fmt.Errorf("item must be a pointer")
	}

	entry, err := b.Builder.Build(b.Item)
	if err != nil {
		return nil, fmt.Errorf("could not built entry for transact put item: %w", err)
	}

	item := &dynamodb.TransactWriteItem{
		Put: &dynamodb.Put{
			ConditionExpression:                 entry.ConditionExpression,
			ExpressionAttributeNames:            entry.ExpressionAttributeNames,
			ExpressionAttributeValues:           entry.ExpressionAttributeValues,
			Item:                                entry.Item,
			TableName:                           entry.TableName,
			ReturnValuesOnConditionCheckFailure: entry.ReturnValues,
		},
	}

	return item, nil
}

func (b *TransactPutItem) GetItem() interface{} {
	return b.Item
}
