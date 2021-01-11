package ddb

import (
	"fmt"
	"github.com/aws/aws-sdk-go/service/dynamodb"
)

//go:generate mockery -name TransactGetItemBuilder
type TransactGetItemBuilder interface {
	Build() (*dynamodb.TransactGetItem, error)
	GetItem() interface{}
}

type TransactGetItem struct {
	Builder GetItemBuilder
	Item    interface{}
}

func (b *TransactGetItem) Build() (*dynamodb.TransactGetItem, error) {
	if !isPointer(b.Item) {
		return nil, fmt.Errorf("item must be a pointer")
	}

	entry, err := b.Builder.Build(b.Item)
	if err != nil {
		return nil, fmt.Errorf("could not built entry for transact get item: %w", err)
	}

	item := &dynamodb.TransactGetItem{
		Get: &dynamodb.Get{
			ExpressionAttributeNames: entry.ExpressionAttributeNames,
			Key:                      entry.Key,
			ProjectionExpression:     entry.ProjectionExpression,
			TableName:                entry.TableName,
		},
	}

	return item, nil
}

func (b *TransactGetItem) GetItem() interface{} {
	return b.Item
}
