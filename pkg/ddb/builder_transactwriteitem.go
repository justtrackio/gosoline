package ddb

import (
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

//go:generate go run github.com/vektra/mockery/v2 --name TransactWriteItemBuilder
type TransactWriteItemBuilder interface {
	Build() (*types.TransactWriteItem, error)
	GetItem() interface{}
}
