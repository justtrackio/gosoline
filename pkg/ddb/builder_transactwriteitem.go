package ddb

import "github.com/aws/aws-sdk-go/service/dynamodb"

//go:generate mockery --name TransactWriteItemBuilder
type TransactWriteItemBuilder interface {
	Build() (*dynamodb.TransactWriteItem, error)
	GetItem() interface{}
}
