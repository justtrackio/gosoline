package ddb

import (
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

type TransactPutItem struct {
	Builder PutItemBuilder
	Item    any
}

func NewTransactionPutItemBuilder() *TransactPutItem {
	return &TransactPutItem{}
}

func (b *TransactPutItem) Build() (*types.TransactWriteItem, error) {
	if !isPointer(b.Item) {
		return nil, fmt.Errorf("item must be a pointer")
	}

	entry, err := b.Builder.Build(b.Item)
	if err != nil {
		return nil, fmt.Errorf("could not built entry for transact put item: %w", err)
	}

	item := &types.TransactWriteItem{
		Put: &types.Put{
			ConditionExpression:                 entry.ConditionExpression,
			ExpressionAttributeNames:            entry.ExpressionAttributeNames,
			ExpressionAttributeValues:           entry.ExpressionAttributeValues,
			Item:                                entry.Item,
			TableName:                           entry.TableName,
			ReturnValuesOnConditionCheckFailure: types.ReturnValuesOnConditionCheckFailure(entry.ReturnValues),
		},
	}

	return item, nil
}

func (b *TransactPutItem) GetItem() any {
	return b.Item
}
