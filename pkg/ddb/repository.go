package ddb

import (
	"context"
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/clock"
	"github.com/applike/gosoline/pkg/cloud/aws"
	"github.com/applike/gosoline/pkg/exec"
	"github.com/applike/gosoline/pkg/log"
	"github.com/applike/gosoline/pkg/mdl"
	"github.com/applike/gosoline/pkg/refl"
	"github.com/applike/gosoline/pkg/tracing"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbiface"
	"github.com/cenkalti/backoff"
	"github.com/hashicorp/go-multierror"
	"time"
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

	Create = "create"
	Update = "update"
	Delete = "delete"
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
	logger   log.Logger
	tracer   tracing.Tracer
	client   dynamodbiface.DynamoDBAPI
	executor aws.Executor
	clock    clock.Clock

	keyBuilder keyBuilder
	metadata   *Metadata
	settings   *Settings
}

func NewRepository(config cfg.Config, logger log.Logger, settings *Settings) (Repository, error) {
	if settings.ModelId.Name == "" {
		settings.ModelId.Name = getTypeName(settings.Main.Model)
	}

	settings.ModelId.PadFromConfig(config)
	settings.AutoCreate = config.GetBool("aws_dynamoDb_autoCreate")
	settings.Client.MaxRetries = config.GetInt("aws_sdk_retries")

	tableName := TableName(settings)
	client := ProvideClient(config, logger, settings)

	backoffSettings := &exec.BackoffSettings{}
	config.UnmarshalKey("ddb.backoff", backoffSettings)

	if err := cfg.Merge(&settings.Backoff, *backoffSettings); err != nil {
		return nil, fmt.Errorf("could not merge backoff settings for ddb table %s: %w", tableName, err)
	}

	res := &exec.ExecutableResource{
		Type: "ddb",
		Name: tableName,
	}
	executor := aws.NewExecutor(logger, res, &settings.Backoff, func(result interface{}, err error) exec.ErrorType {
		if isError(err, dynamodb.ErrCodeConditionalCheckFailedException) {
			return exec.ErrorTypeOk
		}

		return exec.ErrorTypeUnknown
	})

	svc := NewServiceWithInterfaces(logger, client)
	_, err := svc.CreateTable(settings)

	if err != nil {
		return nil, fmt.Errorf("could not create ddb table %s: %w", tableName, err)
	}

	tracer := tracing.NewNoopTracer()

	if !settings.DisableTracing {
		tracer, err = tracing.ProvideTracer(config, logger)
		if err != nil {
			return nil, fmt.Errorf("can not create tracer: %w", err)
		}
	}

	return NewWithInterfaces(logger, tracer, client, executor, settings)
}

func NewWithInterfaces(logger log.Logger, tracer tracing.Tracer, client dynamodbiface.DynamoDBAPI, executor aws.Executor, settings *Settings) (Repository, error) {
	metadataFactory := NewMetadataFactory()
	metadata, err := metadataFactory.GetMetadata(settings)

	if err != nil {
		name := TableName(settings)
		return nil, fmt.Errorf("could not factor metadata for ddb table %s: %w", name, err)
	}

	keyBuilder := keyBuilder{
		metadata: metadata.Main,
	}

	return &repository{
		logger:     logger,
		tracer:     tracer,
		client:     client,
		executor:   executor,
		keyBuilder: keyBuilder,
		metadata:   metadata,
		settings:   settings,
		clock:      clock.Provider,
	}, nil
}

func (r *repository) GetModelId() mdl.ModelId {
	return r.settings.ModelId
}

func (r *repository) BatchGetItems(ctx context.Context, qb BatchGetItemsBuilder, items interface{}) (*OperationResult, error) {
	_, span := r.tracer.StartSubSpan(ctx, "ddb.BatchGetItems")
	defer span.Finish()

	unmarshaller, err := NewUnmarshallerFromPtrSlice(items)

	if err != nil {
		return nil, fmt.Errorf("can not initialize unmarshaller for BatchGetItems operation on table %s: %w", r.metadata.TableName, err)
	}

	input, err := qb.Build(items)
	result := newOperationResult()

	if err != nil {
		return nil, fmt.Errorf("can not build input for BatchGetItems operation on table %s: %w", r.metadata.TableName, err)
	}

	for input.RequestItems != nil {
		outI, err := r.executor.Execute(ctx, func() (*request.Request, interface{}) {
			return r.client.BatchGetItemRequest(input)
		})

		if exec.IsRequestCanceled(err) {
			return nil, exec.RequestCanceledError
		}

		if isError(err, dynamodb.ErrCodeResourceNotFoundException) {
			return nil, NewTableNotFoundError(r.metadata.TableName, err)
		}

		if err != nil {
			return nil, fmt.Errorf("could not execute BatchGetItems operation for table %s: %w", r.metadata.TableName, err)
		}

		input.RequestItems, err = r.processBatchReadItemsResponse(qb, outI.(*dynamodb.BatchGetItemOutput), unmarshaller, result)

		if err != nil {
			return nil, err
		}
	}

	return result, nil
}

func (r *repository) processBatchReadItemsResponse(qb BatchGetItemsBuilder, out *dynamodb.BatchGetItemOutput, unmarshaller *Unmarshaller, result *OperationResult) (map[string]*dynamodb.KeysAndAttributes, error) {
	responses, err := r.filterResponses(qb, out.Responses[r.metadata.TableName])

	if err != nil {
		return nil, err
	}

	err = unmarshaller.Append(responses)

	if err != nil {
		return nil, fmt.Errorf("could not unmarshal items after BatchGetItems operation for table %s: %w", r.metadata.TableName, err)
	}

	result.ConsumedCapacity.addSlice(out.ConsumedCapacity)

	if _, ok := out.UnprocessedKeys[r.metadata.TableName]; !ok {
		return nil, nil
	}

	return out.UnprocessedKeys, nil
}

func (r *repository) filterResponses(qb BatchGetItemsBuilder, responses []map[string]*dynamodb.AttributeValue) ([]map[string]*dynamodb.AttributeValue, error) {
	var ok bool
	var filterer ttlFilterer

	if filterer, ok = qb.(ttlFilterer); !ok {
		return responses, nil
	}

	filteredResponses := make([]map[string]*dynamodb.AttributeValue, 0, len(responses))

	for _, response := range responses {
		keep, err := filterer.PerformFilterCondition(response)

		if err != nil {
			return nil, fmt.Errorf("could not perform filter condition for table %s: %w", r.metadata.TableName, err)
		}

		if keep {
			filteredResponses = append(filteredResponses, response)
		}
	}

	return filteredResponses, nil
}

func (r *repository) BatchPutItems(ctx context.Context, value interface{}) (*OperationResult, error) {
	_, span := r.tracer.StartSubSpan(ctx, "ddb.BatchPutItems")
	defer span.Finish()

	return r.batchWriteItem(ctx, value, func(item interface{}) (*dynamodb.WriteRequest, error) {
		marshalledItem, err := dynamodbattribute.MarshalMap(item)

		if err != nil {
			return nil, fmt.Errorf("could not marshal item for batchWriteItem operation on table %s: %w", r.metadata.TableName, err)
		}

		return &dynamodb.WriteRequest{
			PutRequest: &dynamodb.PutRequest{
				Item: marshalledItem,
			},
		}, nil
	})
}

func (r *repository) BatchDeleteItems(ctx context.Context, value interface{}) (*OperationResult, error) {
	_, span := r.tracer.StartSubSpan(ctx, "ddb.BatchDeleteItems")
	defer span.Finish()

	return r.batchWriteItem(ctx, value, func(item interface{}) (*dynamodb.WriteRequest, error) {
		key, err := r.keyBuilder.fromItem(item)

		if err != nil {
			return nil, fmt.Errorf("could not create key for item for BatchDeleteItems operation on table %s: %w", r.metadata.TableName, err)
		}

		return &dynamodb.WriteRequest{
			DeleteRequest: &dynamodb.DeleteRequest{
				Key: key,
			},
		}, nil
	})
}

func (r *repository) batchWriteItem(ctx context.Context, value interface{}, reqBuilder func(interface{}) (*dynamodb.WriteRequest, error)) (*OperationResult, error) {
	items, err := refl.InterfaceToInterfaceSlice(value)

	if err != nil {
		return nil, fmt.Errorf("no slice of items provided for batchWriteItem operation on table %s: %w", r.metadata.TableName, err)
	}

	// DynamoDB limits the number of operations per batch request to 25
	chunks := chunk(items, 25)
	result := newOperationResult()

	for _, chunk := range chunks {
		requests := make([]*dynamodb.WriteRequest, len(chunk))

		for i, item := range chunk {
			requests[i], err = reqBuilder(item)
			if err != nil {
				return nil, fmt.Errorf("could not create partial request for batchWriteItem operation on table %s: %w", r.metadata.TableName, err)
			}
		}

		input := &dynamodb.BatchWriteItemInput{
			RequestItems: map[string][]*dynamodb.WriteRequest{
				r.metadata.TableName: requests,
			},
		}

		err := r.chunkWriteItem(ctx, input, result)

		if err != nil {
			return nil, fmt.Errorf("could not write chunk for batchWriteItem operation on table %s: %w", r.metadata.TableName, err)
		}
	}

	return result, nil
}

func (r *repository) chunkWriteItem(ctx context.Context, input *dynamodb.BatchWriteItemInput, result *OperationResult) error {
	backoffConfig := backoff.NewExponentialBackOff()
	backoffConfig.MaxElapsedTime = time.Minute
	backoffConfig.InitialInterval = 100 * time.Millisecond

	finalErr := fmt.Errorf("could not write unprocessed items in chunkWriteItem on table %s", r.metadata.TableName)

	return backoff.Retry(func() error {
		outI, err := r.executor.Execute(ctx, func() (*request.Request, interface{}) {
			return r.client.BatchWriteItemRequest(input)
		})

		if err != nil {
			return backoff.Permanent(fmt.Errorf("could not execute item for batchWriteItemWithContext operation on table %s: %w", r.metadata.TableName, err))
		}

		out := outI.(*dynamodb.BatchWriteItemOutput)
		result.ConsumedCapacity.addSlice(out.ConsumedCapacity)

		if _, ok := out.UnprocessedItems[r.metadata.TableName]; !ok {
			return nil
		}

		processedItems := totalItemCount(input.RequestItems) - totalItemCount(out.UnprocessedItems)
		input.RequestItems = out.UnprocessedItems

		// If we made any process, we try again and reset our backoff. As long as we are making process we can try again
		// and will eventually finish
		if processedItems > 0 {
			backoffConfig.Reset()
		}

		// If we made any progress, this will sleep for a short amount of time and then retry
		// If we did not make any progress, we will sleep for increasingly longer times until
		// we will finally return this error
		return finalErr
	}, backoff.WithContext(backoffConfig, ctx))
}

func totalItemCount(requests map[string][]*dynamodb.WriteRequest) int {
	result := 0

	for _, item := range requests {
		result += len(item)
	}

	return result
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
		return nil, fmt.Errorf("could not build input for DeleteItem operation on table %s: %w", r.metadata.TableName, err)
	}

	outI, err := r.executor.Execute(ctx, func() (*request.Request, interface{}) {
		return r.client.DeleteItemRequest(input)
	})

	if exec.IsRequestCanceled(err) {
		return nil, exec.RequestCanceledError
	}

	if isError(err, dynamodb.ErrCodeResourceNotFoundException) {
		return nil, NewTableNotFoundError(r.metadata.TableName, err)
	}

	if err != nil && !isError(err, dynamodb.ErrCodeConditionalCheckFailedException) {
		return nil, fmt.Errorf("could not execute DeleteItem operation for table %s: %w", r.metadata.TableName, err)
	}

	out := outI.(*dynamodb.DeleteItemOutput)
	result.ConditionalCheckFailed = isError(err, dynamodb.ErrCodeConditionalCheckFailedException)
	result.ConsumedCapacity.add(out.ConsumedCapacity)

	if out.Attributes == nil {
		return result, nil
	}

	err = dynamodbattribute.UnmarshalMap(out.Attributes, item)

	if err != nil {
		return nil, fmt.Errorf("could not unmarshal old value after DeleteItem operation on table %s: %w", r.metadata.TableName, err)
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
		return nil, fmt.Errorf("could not build GetItem expression for table %s: %w", r.metadata.TableName, err)
	}

	outI, err := r.executor.Execute(ctx, func() (*request.Request, interface{}) {
		return r.client.GetItemRequest(input)
	})

	if exec.IsRequestCanceled(err) {
		return nil, exec.RequestCanceledError
	}

	if isError(err, dynamodb.ErrCodeResourceNotFoundException) {
		return nil, NewTableNotFoundError(r.metadata.TableName, err)
	}

	if err != nil {
		return nil, err
	}

	out := outI.(*dynamodb.GetItemOutput)
	result.ConsumedCapacity.add(out.ConsumedCapacity)

	if out.Item == nil {
		return result, nil
	}

	if ttlFilterer, ok := qb.(ttlFilterer); ok {
		keep, err := ttlFilterer.PerformFilterCondition(out.Item)

		if err != nil {
			return nil, err
		}

		if !keep {
			return result, nil
		}
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
		return nil, fmt.Errorf("could not build input and expr for PutItem operation on table %s: %w", r.metadata.TableName, err)
	}

	result := newPutItemResult()

	outI, err := r.executor.Execute(ctx, func() (*request.Request, interface{}) {
		return r.client.PutItemRequest(input)
	})

	if exec.IsRequestCanceled(err) {
		return nil, exec.RequestCanceledError
	}

	if isError(err, dynamodb.ErrCodeResourceNotFoundException) {
		return nil, NewTableNotFoundError(r.metadata.TableName, err)
	}

	if err != nil && !isError(err, dynamodb.ErrCodeConditionalCheckFailedException) {
		return nil, fmt.Errorf("could not execute PutItem operation for table %s: %w", r.metadata.TableName, err)
	}

	out := outI.(*dynamodb.PutItemOutput)
	result.ConditionalCheckFailed = isError(err, dynamodb.ErrCodeConditionalCheckFailedException)
	result.ConsumedCapacity.add(out.ConsumedCapacity)

	if out.Attributes == nil {
		result.IsReturnEmpty = true

		return result, nil
	}

	err = dynamodbattribute.UnmarshalMap(out.Attributes, item)

	if err != nil {
		return nil, fmt.Errorf("could not unmarshal old value after PutItem operation on table %s: %w", r.metadata.TableName, err)
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
	if op.iterator.isDone() {
		return &readResult{}, nil
	}

	outI, err := r.executor.Execute(ctx, func() (*request.Request, interface{}) {
		return r.client.QueryRequest(op.input)
	})

	if exec.IsRequestCanceled(err) {
		return nil, exec.RequestCanceledError
	}

	if isError(err, dynamodb.ErrCodeResourceNotFoundException) {
		return nil, NewTableNotFoundError(r.metadata.TableName, err)
	}

	if err != nil {
		return nil, fmt.Errorf("could not execute Query operation for table %s: %w", r.metadata.TableName, err)
	}

	out := outI.(*dynamodb.QueryOutput)
	op.result.RequestCount++
	op.result.ItemCount += *out.Count
	op.result.ScannedCount += *out.ScannedCount
	op.result.ConsumedCapacity.add(out.ConsumedCapacity)

	nextPageSize := op.iterator.advance(out.Count)

	op.input.Limit = nextPageSize
	op.input.ExclusiveStartKey = out.LastEvaluatedKey

	resp := &readResult{
		Items:            out.Items,
		LastEvaluatedKey: out.LastEvaluatedKey,
		Progress:         op.result,
	}

	return resp, nil
}

func (r *repository) UpdateItem(ctx context.Context, ub UpdateItemBuilder, item interface{}) (*UpdateItemResult, error) {
	_, span := r.tracer.StartSubSpan(ctx, "ddb.UpdateItem")
	defer span.Finish()

	input, err := ub.Build(item)

	if err != nil {
		return nil, fmt.Errorf("could not build input for UpdateItem operation on table %s: %w", r.metadata.TableName, err)
	}

	result := newUpdateItemResult()
	outI, err := r.executor.Execute(ctx, func() (*request.Request, interface{}) {
		return r.client.UpdateItemRequest(input)
	})

	if exec.IsRequestCanceled(err) {
		return nil, exec.RequestCanceledError
	}

	if isError(err, dynamodb.ErrCodeResourceNotFoundException) {
		return nil, NewTableNotFoundError(r.metadata.TableName, err)
	}

	if err != nil && !isError(err, dynamodb.ErrCodeConditionalCheckFailedException) {
		return nil, fmt.Errorf("could not execute UpdateItem operation for table %s: %w", r.metadata.TableName, err)
	}

	out := outI.(*dynamodb.UpdateItemOutput)
	result.ConditionalCheckFailed = isError(err, dynamodb.ErrCodeConditionalCheckFailedException)
	result.ConsumedCapacity.add(out.ConsumedCapacity)

	if out.Attributes == nil {
		return result, nil
	}

	err = dynamodbattribute.UnmarshalMap(out.Attributes, item)

	if err != nil {
		return nil, fmt.Errorf("could not unmarshal old value after UpdateItem operation on table %s: %w", r.metadata.TableName, err)
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
		return nil, fmt.Errorf("can not build scan operation: %w", err)
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
	if op.iterator.isDone() {
		return &readResult{}, nil
	}

	outI, err := r.executor.Execute(ctx, func() (*request.Request, interface{}) {
		return r.client.ScanRequest(op.input)
	})

	if exec.IsRequestCanceled(err) {
		return nil, exec.RequestCanceledError
	}

	if isError(err, dynamodb.ErrCodeResourceNotFoundException) {
		return nil, NewTableNotFoundError(r.metadata.TableName, err)
	}

	if err != nil {
		return nil, err
	}

	out := outI.(*dynamodb.ScanOutput)
	op.result.RequestCount++
	op.result.ItemCount += *out.Count
	op.result.ScannedCount += *out.ScannedCount
	op.result.ConsumedCapacity.add(out.ConsumedCapacity)

	nextPageSize := op.iterator.advance(out.Count)

	op.input.Limit = nextPageSize
	op.input.ExclusiveStartKey = out.LastEvaluatedKey

	return &readResult{
		Items:            out.Items,
		LastEvaluatedKey: out.LastEvaluatedKey,
		Progress:         op.result,
	}, nil
}

func (r *repository) BatchGetItemsBuilder() BatchGetItemsBuilder {
	return NewBatchGetItemsBuilder(r.metadata, r.clock)
}

func (r *repository) DeleteItemBuilder() DeleteItemBuilder {
	return NewDeleteItemBuilder(r.metadata)
}

func (r *repository) GetItemBuilder() GetItemBuilder {
	return NewGetItemBuilder(r.metadata, r.clock)
}

func (r *repository) PutItemBuilder() PutItemBuilder {
	return NewPutItemBuilder(r.metadata)
}

func (r *repository) QueryBuilder() QueryBuilder {
	return NewQueryBuilder(r.metadata, r.clock)
}

func (r *repository) ScanBuilder() ScanBuilder {
	return NewScanBuilder(r.metadata, r.clock)
}

func (r *repository) UpdateItemBuilder() UpdateItemBuilder {
	return NewUpdateItemBuilder(r.metadata)
}

func (r *repository) readAll(items interface{}, read func() (*readResult, error)) error {
	unmarshaller, err := NewUnmarshallerFromPtrSlice(items)

	if err != nil {
		return fmt.Errorf("can not initialize unmarshaller for operation on table %s: %w", r.metadata.TableName, err)
	}

	for {
		out, err := read()

		if err != nil {
			return fmt.Errorf("could not execute read operation for table %s: %w", r.metadata.TableName, err)
		}

		if out.Items == nil {
			break
		}

		err = unmarshaller.Append(out.Items)

		if err != nil {
			return fmt.Errorf("could not unmarshal items after Query operation for table %s: %w", r.metadata.TableName, err)
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
		return fmt.Errorf("can not initialize unmarshaller for operation on table %s: %w", r.metadata.TableName, err)
	}

	var callbackErrors error

	for {
		out, err := read()

		if err != nil {
			return fmt.Errorf("could not execute read operation for table %s: %w", r.metadata.TableName, err)
		}

		if out.Items == nil || len(out.Items) == 0 {
			return callbackErrors
		}

		items, err := unmarshaller.Unmarshal(out.Items)

		if err != nil {
			return fmt.Errorf("could not unmarshal items after read operation for table %s: %w", r.metadata.TableName, err)
		}

		cont, err := callback(ctx, items, out.Progress)

		if err == nil && !cont {
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
