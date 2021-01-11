package ddb

import (
	"fmt"
	"github.com/aws/aws-sdk-go/service/dynamodb"
)

type TransactConditionCheck struct {
	Builder ConditionCheckBuilder
	Item    interface{}
}

func (b *TransactConditionCheck) Build() (*dynamodb.TransactWriteItem, error) {
	if !isPointer(b.Item) {
		return nil, fmt.Errorf("item must be a pointer")
	}

	entry, err := b.Builder.Build(b.Item)
	if err != nil {
		return nil, fmt.Errorf("could not built entry for transact condition check: %w", err)
	}

	item := &dynamodb.TransactWriteItem{
		ConditionCheck: entry,
	}

	return item, nil
}

func (b *TransactConditionCheck) GetItem() interface{} {
	return b.Item
}
