package ddb

import (
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

type TransactConditionCheck struct {
	Builder ConditionCheckBuilder
	Item    any
}

func (b *TransactConditionCheck) Build() (*types.TransactWriteItem, error) {
	if !isPointer(b.Item) {
		return nil, fmt.Errorf("item must be a pointer")
	}

	entry, err := b.Builder.Build(b.Item)
	if err != nil {
		return nil, fmt.Errorf("could not built entry for transact condition check: %w", err)
	}

	item := &types.TransactWriteItem{
		ConditionCheck: entry,
	}

	return item, nil
}

func (b *TransactConditionCheck) GetItem() any {
	return b.Item
}
