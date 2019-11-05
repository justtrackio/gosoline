package ddb

import (
	"context"
	"github.com/aws/aws-sdk-go/service/dynamodb"
)

type ResultCallback func(ctx context.Context, result interface{}) (bool, error)

type readResult struct {
	Items            []map[string]*dynamodb.AttributeValue
	LastEvaluatedKey map[string]*dynamodb.AttributeValue
}

type OperationResult struct {
	ConsumedCapacity *ConsumedCapacity
}

func newOperationResult() *OperationResult {
	return &OperationResult{
		ConsumedCapacity: newConsumedCapacity(),
	}
}

type DeleteItemResult struct {
	ConditionalCheckFailed bool
	ConsumedCapacity       *ConsumedCapacity
}

func newDeleteItemResult() *DeleteItemResult {
	return &DeleteItemResult{
		ConsumedCapacity: newConsumedCapacity(),
	}
}

type GetItemResult struct {
	IsFound          bool
	ConsumedCapacity *ConsumedCapacity
}

func newGetItemResult() *GetItemResult {
	return &GetItemResult{
		ConsumedCapacity: newConsumedCapacity(),
	}
}

type PutItemResult struct {
	ConditionalCheckFailed bool
	ConsumedCapacity       *ConsumedCapacity
	IsReturnEmpty          bool
}

func newPutItemResult() *PutItemResult {
	return &PutItemResult{
		ConsumedCapacity: newConsumedCapacity(),
	}
}

type QueryResult struct {
	RequestCount     int64
	ItemCount        int64
	ScannedCount     int64
	ConsumedCapacity *ConsumedCapacity
}

func newQueryResult() *QueryResult {
	return &QueryResult{
		ConsumedCapacity: newConsumedCapacity(),
	}
}

type ScanResult struct {
	RequestCount     int64
	ItemCount        int64
	ScannedCount     int64
	ConsumedCapacity *ConsumedCapacity
}

func newScanResult() *ScanResult {
	return &ScanResult{
		ConsumedCapacity: newConsumedCapacity(),
	}
}

type UpdateItemResult struct {
	ConditionalCheckFailed bool
	ConsumedCapacity       *ConsumedCapacity
}

func newUpdateItemResult() *UpdateItemResult {
	return &UpdateItemResult{
		ConsumedCapacity: newConsumedCapacity(),
	}
}
