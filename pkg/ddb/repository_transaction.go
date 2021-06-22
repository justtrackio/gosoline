package ddb

import (
	"context"
	"errors"
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/cloud/aws"
	"github.com/applike/gosoline/pkg/exec"
	"github.com/applike/gosoline/pkg/log"
	"github.com/applike/gosoline/pkg/tracing"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbiface"
	"github.com/hashicorp/go-multierror"
)

//go:generate mockery -name TransactionRepository
type TransactionRepository interface {
	TransactWriteItems(ctx context.Context, items []TransactWriteItemBuilder) (*OperationResult, error)
	TransactGetItems(ctx context.Context, items []TransactGetItemBuilder) (*OperationResult, error)
}

type transactionRepository struct {
	logger log.Logger

	client   dynamodbiface.DynamoDBAPI
	executor aws.Executor
	tracer   tracing.Tracer
}

func NewTransactionRepository(config cfg.Config, logger log.Logger) (*transactionRepository, error) {
	settings := &Settings{}

	settings.Client.MaxRetries = config.GetInt("aws_sdk_retries")

	backoffSettings := &exec.BackoffSettings{}
	config.UnmarshalKey("ddb.backoff", backoffSettings)

	if err := cfg.Merge(&settings.Backoff, *backoffSettings); err != nil {
		return nil, fmt.Errorf("could not merge backoff settings for transactions: %w", err)
	}

	tracer, err := tracing.ProvideTracer(config, logger)
	if err != nil {
		return nil, fmt.Errorf("can not create tracer: %w", err)
	}

	client := ProvideClient(config, logger, settings)

	res := &exec.ExecutableResource{
		Type: "ddb",
		Name: "transaction",
	}

	checks := []exec.ErrorChecker{
		checkPreconditionFailed,
		checkTransactionConflict,
	}

	executor := aws.NewExecutor(logger, res, &settings.Backoff, checks...)

	return NewTransactionRepositoryWithInterfaces(logger, client, executor, tracer), nil
}

func NewTransactionRepositoryWithInterfaces(logger log.Logger, client dynamodbiface.DynamoDBAPI, executor aws.Executor, tracer tracing.Tracer) *transactionRepository {
	return &transactionRepository{
		logger:   logger,
		client:   client,
		executor: executor,
		tracer:   tracer,
	}
}

func (r transactionRepository) TransactGetItems(ctx context.Context, items []TransactGetItemBuilder) (*OperationResult, error) {
	res := newOperationResult()

	if len(items) == 0 {
		return res, nil
	}

	_, span := r.tracer.StartSubSpan(ctx, "ddb.TransactGetItems")
	defer span.Finish()

	var err error
	transactionItems := make([]*dynamodb.TransactGetItem, len(items))

	for i, v := range items {
		transactionItems[i], err = v.Build()
		if err != nil {
			return nil, err
		}
	}

	input := &dynamodb.TransactGetItemsInput{
		TransactItems: transactionItems,
	}

	outI, err := r.executor.Execute(ctx, func() (*request.Request, interface{}) {
		return r.client.TransactGetItemsRequest(input)
	})

	if exec.IsRequestCanceled(err) {
		return nil, exec.RequestCanceledError
	}

	// TODO check error response doc
	if err != nil {
		return nil, parseTransactionError(err)
	}

	out := outI.(*dynamodb.TransactGetItemsOutput)
	res.ConsumedCapacity.addSlice(out.ConsumedCapacity)

	for i, itemResponse := range out.Responses {
		err = dynamodbattribute.UnmarshalMap(itemResponse.Item, items[i].GetItem())
		if err != nil {
			return nil, fmt.Errorf("could not unmarshal partial response: %w", err)
		}
	}

	return res, nil
}

func (r transactionRepository) TransactWriteItems(ctx context.Context, itemBuilders []TransactWriteItemBuilder) (*OperationResult, error) {
	return r.TransactWriteItemsIdempotent(ctx, itemBuilders, nil)
}

// ClientRequestToken enforces idempotency over a ten minute time frame
// https://docs.aws.amazon.com/amazondynamodb/latest/APIReference/API_TransactWriteItems.html#DDB-TransactWriteItems-request-ClientRequestToken
func (r transactionRepository) TransactWriteItemsIdempotent(ctx context.Context, itemBuilders []TransactWriteItemBuilder, clientRequestToken *string) (*OperationResult, error) {
	res := newOperationResult()

	if len(itemBuilders) == 0 {
		return res, nil
	}

	_, span := r.tracer.StartSubSpan(ctx, "ddb.TransactWriteItems")
	defer span.Finish()

	transactionItems := make([]*dynamodb.TransactWriteItem, 0)
	for _, v := range itemBuilders {
		item, err := v.Build()
		if err != nil {
			return nil, err
		}

		transactionItems = append(transactionItems, item)
	}

	input := dynamodb.TransactWriteItemsInput{
		ClientRequestToken: clientRequestToken,
		TransactItems:      transactionItems,
	}

	outI, err := r.executor.Execute(ctx, func() (*request.Request, interface{}) {
		return r.client.TransactWriteItemsRequest(&input)
	})

	if exec.IsRequestCanceled(err) {
		return nil, exec.RequestCanceledError
	}

	if err != nil {
		trcErr := &dynamodb.TransactionCanceledException{}

		if errors.As(err, &trcErr) {
			return nil, transformTransactionCanceledError(trcErr, itemBuilders)
		}

		return nil, parseTransactionError(err)
	}

	out := (outI).(*dynamodb.TransactWriteItemsOutput)
	res.ConsumedCapacity.addSlice(out.ConsumedCapacity)

	return res, parseTransactionError(err)
}

func transformTransactionCanceledError(tcErr *dynamodb.TransactionCanceledException, itemBuilders []TransactWriteItemBuilder) error {
	multiErr := multierror.Append(&multierror.Error{}, parseTransactionError(tcErr))

	for i, reason := range tcErr.CancellationReasons {
		if *reason.Code != cancellationReasonConditionCheckFailed {
			continue
		}

		err := dynamodbattribute.UnmarshalMap(reason.Item, itemBuilders[i].GetItem())
		if err != nil {
			unmarshalErr := fmt.Errorf("could not unmarshal partial response: %w", err)
			multiErr = multierror.Append(multiErr, unmarshalErr)
		}
	}

	return multiErr.ErrorOrNil()
}
