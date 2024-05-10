package ddb

import (
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

//go:generate go run github.com/vektra/mockery/v2 --name TransactGetItemBuilder
type TransactGetItemBuilder interface {
	Build() (types.TransactGetItem, error)
	GetItem() any
}

type TransactGetItem struct {
	Builder GetItemBuilder
	Item    any
}

func (b *TransactGetItem) Build() (types.TransactGetItem, error) {
	if !isPointer(b.Item) {
		return types.TransactGetItem{}, fmt.Errorf("item must be a pointer")
	}

	entry, err := b.Builder.Build(b.Item)
	if err != nil {
		return types.TransactGetItem{}, fmt.Errorf("could not built entry for transact get item: %w", err)
	}

	item := types.TransactGetItem{
		Get: &types.Get{
			ExpressionAttributeNames: entry.ExpressionAttributeNames,
			Key:                      entry.Key,
			ProjectionExpression:     entry.ProjectionExpression,
			TableName:                entry.TableName,
		},
	}

	return item, nil
}

func (b *TransactGetItem) GetItem() any {
	return b.Item
}
