package ddb

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

type ResultCallback func(ctx context.Context, items interface{}, progress Progress) (bool, error)

type Progress interface {
	GetRequestCount() int32
	GetItemCount() int32
	GetScannedCount() int32
	GetConsumedCapacity() *ConsumedCapacity
}

type readResult struct {
	Items            []map[string]types.AttributeValue
	LastEvaluatedKey map[string]types.AttributeValue
	Progress         Progress
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
	RequestCount     int32
	ItemCount        int32
	ScannedCount     int32
	ConsumedCapacity *ConsumedCapacity
}

func (q QueryResult) GetRequestCount() int32 {
	return q.RequestCount
}

func (q QueryResult) GetItemCount() int32 {
	return q.ItemCount
}

func (q QueryResult) GetScannedCount() int32 {
	return q.ScannedCount
}

func (q QueryResult) GetConsumedCapacity() *ConsumedCapacity {
	return q.ConsumedCapacity
}

func newQueryResult() *QueryResult {
	return &QueryResult{
		ConsumedCapacity: newConsumedCapacity(),
	}
}

type ScanResult struct {
	RequestCount     int32
	ItemCount        int32
	ScannedCount     int32
	ConsumedCapacity *ConsumedCapacity
}

func (s ScanResult) GetRequestCount() int32 {
	return s.RequestCount
}

func (s ScanResult) GetItemCount() int32 {
	return s.ItemCount
}

func (s ScanResult) GetScannedCount() int32 {
	return s.ScannedCount
}

func (s ScanResult) GetConsumedCapacity() *ConsumedCapacity {
	return s.ConsumedCapacity
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
