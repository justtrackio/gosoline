package ddb

import (
	"context"
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/cloud"
	"github.com/applike/gosoline/pkg/mdl"
	"github.com/applike/gosoline/pkg/mon"
	"github.com/applike/gosoline/pkg/tracing"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbiface"
	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
)

const (
	MetricNameAccessSuccess = "DdbAccessSuccess"
	MetricNameAccessFailure = "DdbAccessFailure"
	MetricNameAccessLatency = "DdbAccessLatency"

	OpSave = "save"

	StreamViewTypeNewImage        = dynamodb.StreamViewTypeNewImage
	StreamViewTypeOldImage        = dynamodb.StreamViewTypeOldImage
	StreamViewTypeNewAndOldImages = dynamodb.StreamViewTypeNewAndOldImages
	StreamViewTypeKeysOnly        = dynamodb.StreamViewTypeKeysOnly
)

//go:generate mockery -name Repository
type Repository interface {
	GetModelId() mdl.ModelId

	BatchDeleteItems(ctx context.Context, value interface{}) (*OperationResult, error)
	BatchGetItems(ctx context.Context, qb BatchGetItemsBuilder, result interface{}) (*OperationResult, error)
	BatchPutItems(ctx context.Context, items interface{}) (*OperationResult, error)
	DeleteItem(ctx context.Context, db DeleteItemBuilder, item interface{}) (*DeleteItemResult, error)
	GetItem(ctx context.Context, qb GetItemBuilder, result interface{}) (*GetItemResult, error)
	PutItem(ctx context.Context, qb PutItemBuilder, item interface{}) (*PutItemResult, error)
	Query(ctx context.Context, qb QueryBuilder, result interface{}) (*QueryResult, error)
	Scan(ctx context.Context, sb ScanBuilder, result interface{}) (*ScanResult, error)
	UpdateItem(ctx context.Context, ub UpdateItemBuilder, item interface{}) (*UpdateItemResult, error)

	BatchGetItemsBuilder() BatchGetItemsBuilder
	DeleteItemBuilder() DeleteItemBuilder
	GetItemBuilder() GetItemBuilder
	QueryBuilder() QueryBuilder
	PutItemBuilder() PutItemBuilder
	ScanBuilder() ScanBuilder
	UpdateItemBuilder() UpdateItemBuilder
}

type repository struct {
	logger mon.Logger
	tracer tracing.Tracer
	client dynamodbiface.DynamoDBAPI

	metadata *Metadata
	settings *Settings
}

func NewRepository(config cfg.Config, logger mon.Logger, settings *Settings) *repository {
	if settings.ModelId.Name == "" {
		settings.ModelId.Name = getTypeName(settings.Main.Model)
	}

	settings.ModelId.PadFromConfig(config)
	settings.AutoCreate = config.GetBool("aws_dynamoDb_autoCreate")

	tracer := tracing.NewAwsTracer(config)
	client := cloud.GetDynamoDbClient(config, logger)

	svc := NewService(config, logger)
	_, err := svc.CreateTable(settings)

	if err != nil {
		name := namingStrategy(settings.ModelId)
		logger.Fatalf(err, "could not create ddb table %s", name)
	}

	return NewWithInterfaces(logger, tracer, client, settings)
}

func NewWithInterfaces(logger mon.Logger, tracer tracing.Tracer, client dynamodbiface.DynamoDBAPI, settings *Settings) *repository {
	metadataFactory := NewMetadataFactory()
	metadata, err := metadataFactory.GetMetadata(settings)

	if err != nil {
		name := namingStrategy(settings.ModelId)
		logger.Fatalf(err, "could not factor metadata for ddb table %s", name)
	}

	return &repository{
		logger:   logger,
		tracer:   tracer,
		client:   client,
		metadata: metadata,
		settings: settings,
	}
}

func (r *repository) GetModelId() mdl.ModelId {
	return r.settings.ModelId
}

func (r *repository) BatchGetItems(ctx context.Context, qb BatchGetItemsBuilder, items interface{}) (*OperationResult, error) {
	_, span := r.tracer.StartSubSpan(ctx, "ddb.BatchGetItems")
	defer span.Finish()

	unmarshaller, err := NewUnmarshallerFromPtrSlice(items)

	if err != nil {
		return nil, errors.Wrapf(err, "can not initializer unmarshaller for BatchGetItems operation on table %s", r.metadata.TableName)
	}

	input, err := qb.Build(items)
	result := newOperationResult()

	if err != nil {
		return nil, errors.Wrapf(err, "can not build input for BatchGetItems operation on table %s", r.metadata.TableName)
	}

	for {
		out, err := r.client.BatchGetItemWithContext(ctx, input)

		if err != nil {
			return nil, errors.Wrapf(err, "could not execute BatchGetItems operation for table %s", r.metadata.TableName)
		}

		responses := out.Responses[r.metadata.TableName]
		err = unmarshaller.Append(responses)

		if err != nil {
			return nil, errors.Wrapf(err, "could not unmarshal items after BatchGetItems operation for table %s", r.metadata.TableName)
		}

		result.ConsumedCapacity.addSlice(out.ConsumedCapacity)

		if _, ok := out.UnprocessedKeys[r.metadata.TableName]; !ok {
			break
		}

		input.RequestItems = out.UnprocessedKeys
	}

	return result, nil
}

func (r *repository) BatchPutItems(ctx context.Context, value interface{}) (*OperationResult, error) {
	_, span := r.tracer.StartSubSpan(ctx, "ddb.BatchPutItems")
	defer span.Finish()

	return r.batchWriteItem(ctx, value, func(item map[string]*dynamodb.AttributeValue) *dynamodb.WriteRequest {
		return &dynamodb.WriteRequest{
			PutRequest: &dynamodb.PutRequest{
				Item: item,
			},
		}
	})
}

func (r *repository) BatchDeleteItems(ctx context.Context, value interface{}) (*OperationResult, error) {
	_, span := r.tracer.StartSubSpan(ctx, "ddb.BatchDeleteItems")
	defer span.Finish()

	return r.batchWriteItem(ctx, value, func(item map[string]*dynamodb.AttributeValue) *dynamodb.WriteRequest {
		return &dynamodb.WriteRequest{
			DeleteRequest: &dynamodb.DeleteRequest{
				Key: item,
			},
		}
	})
}

func (r *repository) batchWriteItem(ctx context.Context, value interface{}, reqBuilder func(map[string]*dynamodb.AttributeValue) *dynamodb.WriteRequest) (*OperationResult, error) {
	items, err := interfaceToSliceOfInterfaces(value)

	if err != nil {
		return nil, errors.Wrapf(err, "no slice of items provided for batchWriteItem operation on table %s", r.metadata.TableName)
	}

	chunks := chunk(items, 25)
	result := newOperationResult()

	for _, chunk := range chunks {
		requests := make([]*dynamodb.WriteRequest, len(chunk))

		for i := 0; i < len(chunk); i++ {
			marshalledItem, err := dynamodbattribute.MarshalMap(chunk[i])

			if err != nil {
				return nil, errors.Wrapf(err, "could not marshal item for batchWriteItem operation on table %s", r.metadata.TableName)
			}

			requests[i] = reqBuilder(marshalledItem)
		}

		input := &dynamodb.BatchWriteItemInput{
			RequestItems: map[string][]*dynamodb.WriteRequest{
				r.metadata.TableName: requests,
			},
		}

		err = r.chunkWriteItem(ctx, input, result)

		if err != nil {
			return nil, err
		}
	}

	return result, nil
}

func (r *repository) chunkWriteItem(ctx context.Context, input *dynamodb.BatchWriteItemInput, result *OperationResult) error {
	// try up to 3 times to write the chunk
	for i := 0; i < 3; i++ {
		out, err := r.client.BatchWriteItemWithContext(ctx, input)

		if err != nil {
			return errors.Wrapf(err, "could not execute item for batchWriteItemWithContext operation on table %s", r.metadata.TableName)
		}

		result.ConsumedCapacity.addSlice(out.ConsumedCapacity)

		if _, ok := out.UnprocessedItems[r.metadata.TableName]; !ok {
			return nil
		}

		input.RequestItems = out.UnprocessedItems
	}

	return fmt.Errorf("could not write unprocessed items in chunkWriteItem on table %s", r.metadata.TableName)
}

func (r *repository) DeleteItem(ctx context.Context, db DeleteItemBuilder, item interface{}) (*DeleteItemResult, error) {
	_, span := r.tracer.StartSubSpan(ctx, "ddb.DeleteItem")
	defer span.Finish()

	if db == nil {
		db = r.DeleteItemBuilder()
	}

	input, err := db.Build(item)
	result := newDeleteItemResult()

	if err != nil {
		return nil, errors.Wrapf(err, "could not build input for DeleteItem operation on table [%s]", r.metadata.TableName)
	}

	out, err := r.client.DeleteItemWithContext(ctx, input)

	if err != nil && !isError(err, dynamodb.ErrCodeConditionalCheckFailedException) {
		return nil, errors.Wrapf(err, "could not execute DeleteItem operation for table %s", r.metadata.TableName)
	}

	result.ConditionalCheckFailed = isError(err, dynamodb.ErrCodeConditionalCheckFailedException)
	result.ConsumedCapacity.add(out.ConsumedCapacity)

	if out.Attributes == nil {
		return result, nil
	}

	err = dynamodbattribute.UnmarshalMap(out.Attributes, item)

	if err != nil {
		return nil, errors.Wrapf(err, "could not unmarshal old value after DeleteItem operation on table [%s]", r.metadata.TableName)
	}

	return result, nil
}

func (r *repository) GetItem(ctx context.Context, qb GetItemBuilder, item interface{}) (*GetItemResult, error) {
	_, span := r.tracer.StartSubSpan(ctx, "ddb.GetItem")
	defer span.Finish()

	if qb == nil {
		qb = r.GetItemBuilder()
	}

	input, err := qb.Build(item)
	result := newGetItemResult()

	if err != nil {
		return nil, errors.Wrapf(err, "could not build GetItem expression for table %s", r.metadata.TableName)
	}

	out, err := r.client.GetItemWithContext(ctx, input)

	if err != nil {
		return nil, err
	}

	result.ConsumedCapacity.add(out.ConsumedCapacity)

	if out.Item == nil {
		return result, nil
	}

	result.IsFound = true
	err = dynamodbattribute.UnmarshalMap(out.Item, item)

	if err != nil {
		return nil, err
	}

	return result, nil
}

func (r *repository) PutItem(ctx context.Context, qb PutItemBuilder, item interface{}) (*PutItemResult, error) {
	_, span := r.tracer.StartSubSpan(ctx, "ddb.PutItem")
	defer span.Finish()

	if !isStruct(item) {
		return nil, fmt.Errorf("you have to provice a struct value to PutItem on table [%s] but instead used [%T]", r.metadata.TableName, item)
	}

	if qb == nil {
		qb = r.PutItemBuilder()
	}

	input, err := qb.Build(item)

	if err != nil {
		return nil, errors.Wrapf(err, "could not build input and expr for PutItem operation on table %s", r.metadata.TableName)
	}

	marshaledItem, err := dynamodbattribute.MarshalMap(item)

	if err != nil {
		return nil, errors.Wrapf(err, "could not marshal item for PutItem operation on table %s", r.metadata.TableName)
	}

	input.Item = marshaledItem
	out, err := r.client.PutItemWithContext(ctx, input)
	result := newPutItemResult()

	if err != nil && !isError(err, dynamodb.ErrCodeConditionalCheckFailedException) {
		return nil, errors.Wrapf(err, "could not execute PutItem operation for table %s", r.metadata.TableName)
	}

	result.ConditionalCheckFailed = isError(err, dynamodb.ErrCodeConditionalCheckFailedException)
	result.ConsumedCapacity.add(out.ConsumedCapacity)

	if out.Attributes == nil {
		return result, nil
	}

	err = dynamodbattribute.UnmarshalMap(out.Attributes, item)

	if err != nil {
		return nil, errors.Wrapf(err, "could not unmarshal old value after PutItem operation on table %s", r.metadata.TableName)
	}

	return result, nil
}

func (r *repository) Query(ctx context.Context, qb QueryBuilder, items interface{}) (*QueryResult, error) {
	_, span := r.tracer.StartSubSpan(ctx, "ddb.Query")
	defer span.Finish()

	op, err := qb.Build(items)

	if err != nil {
		return nil, err
	}

	if callback, ok := isResultCallback(items); ok {
		err = r.readCallback(ctx, op.targetType, callback, func() (*readResult, error) {
			return r.doQuery(ctx, op)
		})

		return op.result, err
	}

	err = r.readAll(items, func() (*readResult, error) {
		return r.doQuery(ctx, op)
	})

	return op.result, err
}

func (r *repository) doQuery(ctx context.Context, op *QueryOperation) (*readResult, error) {
	if op.progress.isDone() {
		return &readResult{}, nil
	}

	out, err := r.client.QueryWithContext(ctx, op.input)

	if err != nil {
		return nil, errors.Wrapf(err, "could not execute Query operation for table %s", r.metadata.TableName)
	}

	op.result.RequestCount++
	op.result.ItemCount += *out.Count
	op.result.ScannedCount += *out.ScannedCount
	op.result.ConsumedCapacity.add(out.ConsumedCapacity)

	nextPageSize := op.progress.advance(out.Count)

	op.input.Limit = nextPageSize
	op.input.ExclusiveStartKey = out.LastEvaluatedKey

	resp := &readResult{
		Items:            out.Items,
		LastEvaluatedKey: out.LastEvaluatedKey,
	}

	return resp, nil
}

func (r *repository) UpdateItem(ctx context.Context, ub UpdateItemBuilder, item interface{}) (*UpdateItemResult, error) {
	_, span := r.tracer.StartSubSpan(ctx, "ddb.UpdateItem")
	defer span.Finish()

	input, err := ub.Build(item)

	if err != nil {
		return nil, errors.Wrapf(err, "could not build input for UpdateItem operation on table %s", r.metadata.TableName)
	}

	out, err := r.client.UpdateItemWithContext(ctx, input)
	result := newUpdateItemResult()

	if err != nil && !isError(err, dynamodb.ErrCodeConditionalCheckFailedException) {
		return nil, errors.Wrapf(err, "could not execute UpdateItem operation for table %s", r.metadata.TableName)
	}

	result.ConditionalCheckFailed = isError(err, dynamodb.ErrCodeConditionalCheckFailedException)
	result.ConsumedCapacity.add(out.ConsumedCapacity)

	if out.Attributes == nil {
		return result, nil
	}

	err = dynamodbattribute.UnmarshalMap(out.Attributes, item)

	if err != nil {
		return nil, errors.Wrapf(err, "could not unmarshal old value after UpdateItem operation on table %s", r.metadata.TableName)
	}

	return result, nil
}

func (r *repository) Scan(ctx context.Context, sb ScanBuilder, items interface{}) (*ScanResult, error) {
	_, span := r.tracer.StartSubSpan(ctx, "ddb.Scan")
	defer span.Finish()

	if sb == nil {
		sb = r.ScanBuilder()
	}

	op, err := sb.Build(items)

	if err != nil {
		return nil, err
	}

	if callback, ok := isResultCallback(items); ok {
		err = r.readCallback(ctx, op.targetType, callback, func() (*readResult, error) {
			return r.doScan(ctx, op)
		})

		return op.result, err
	}

	err = r.readAll(items, func() (*readResult, error) {
		return r.doScan(ctx, op)
	})

	return op.result, err
}

func (r *repository) doScan(ctx context.Context, op *ScanOperation) (*readResult, error) {
	if op.progress.isDone() {
		return &readResult{}, nil
	}

	out, err := r.client.ScanWithContext(ctx, op.input)

	if err != nil {
		return nil, err
	}

	op.result.RequestCount++
	op.result.ItemCount += *out.Count
	op.result.ScannedCount += *out.ScannedCount
	op.result.ConsumedCapacity.add(out.ConsumedCapacity)

	nextPageSize := op.progress.advance(out.Count)

	op.input.Limit = nextPageSize
	op.input.ExclusiveStartKey = out.LastEvaluatedKey

	return &readResult{
		Items:            out.Items,
		LastEvaluatedKey: out.LastEvaluatedKey,
	}, nil
}

func (r *repository) BatchGetItemsBuilder() BatchGetItemsBuilder {
	return NewBatchGetItemsBuilder(r.metadata)
}

func (r *repository) DeleteItemBuilder() DeleteItemBuilder {
	return NewDeleteItemBuilder(r.metadata)
}

func (r *repository) GetItemBuilder() GetItemBuilder {
	return NewGetItemBuilder(r.metadata)
}

func (r *repository) PutItemBuilder() PutItemBuilder {
	return NewPutItemBuilder(r.metadata)
}

func (r *repository) QueryBuilder() QueryBuilder {
	return NewQueryBuilder(r.metadata)
}

func (r *repository) ScanBuilder() ScanBuilder {
	return NewScanBuilder(r.metadata)
}

func (r *repository) UpdateItemBuilder() UpdateItemBuilder {
	return NewUpdateItemBuilder(r.metadata)
}

func (r *repository) readAll(items interface{}, read func() (*readResult, error)) error {
	unmarshaller, err := NewUnmarshallerFromPtrSlice(items)

	if err != nil {
		return errors.Wrapf(err, "can not initializer unmarshaller for operation on table %s", r.metadata.TableName)
	}

	for {
		out, err := read()

		if err != nil {
			return errors.Wrapf(err, "could not execute read operation for table %s", r.metadata.TableName)
		}

		if out.Items == nil {
			break
		}

		err = unmarshaller.Append(out.Items)

		if err != nil {
			return errors.Wrapf(err, "could not unmarshal items after Query operation for table %s", r.metadata.TableName)
		}

		if out.LastEvaluatedKey == nil {
			break
		}
	}

	return nil
}

func (r *repository) readCallback(ctx context.Context, items interface{}, callback ResultCallback, read func() (*readResult, error)) error {
	unmarshaller, err := NewUnmarshallerFromStruct(items)

	if err != nil {
		return errors.Wrapf(err, "can not initializer unmarshaller for operation on table %s", r.metadata.TableName)
	}

	var callbackErrors error

	for {
		out, err := read()

		if err != nil {
			return errors.Wrapf(err, "could not execute read operation for table %s", r.metadata.TableName)
		}

		if out.Items == nil || len(out.Items) == 0 {
			return callbackErrors
		}

		result, err := unmarshaller.Unmarshal(out.Items)

		if err != nil {
			return errors.Wrapf(err, "could not unmarshal items after read operation for table %s", r.metadata.TableName)
		}

		cont, err := callback(ctx, result)

		if err != nil && !cont {
			return err
		}

		if err != nil {
			callbackErrors = multierror.Append(callbackErrors, err)
		}

		if out.LastEvaluatedKey == nil {
			break
		}
	}

	return callbackErrors
}

func isError(err error, awsCode string) bool {
	var ok bool
	var aerr awserr.Error

	if err == nil {
		return false
	}

	if aerr, ok = err.(awserr.Error); !ok {
		return false
	}

	if aerr.Code() == awsCode {
		return true
	}

	return false
}
