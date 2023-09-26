package ddb

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/justtrackio/gosoline/pkg/cfg"
	gosoDynamodb "github.com/justtrackio/gosoline/pkg/cloud/aws/dynamodb"
	"github.com/justtrackio/gosoline/pkg/exec"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/tracing"
)

//go:generate mockery --name TransactionRepository
type TransactionRepository interface {
	TransactWriteItems(ctx context.Context, items []TransactWriteItemBuilder) (*OperationResult, error)
	TransactGetItems(ctx context.Context, items []TransactGetItemBuilder) (*OperationResult, error)
}

type transactionRepository struct {
	logger log.Logger
	client gosoDynamodb.Client
	tracer tracing.Tracer
}

func NewTransactionRepository(ctx context.Context, config cfg.Config, logger log.Logger, name string) (*transactionRepository, error) {
	var err error
	var client gosoDynamodb.Client
	var tracer tracing.Tracer

	if client, err = gosoDynamodb.ProvideClient(ctx, config, logger, name); err != nil {
		return nil, fmt.Errorf("can not create dynamodb client: %w", err)
	}

	if tracer, err = tracing.ProvideTracer(ctx, config, logger); err != nil {
		return nil, fmt.Errorf("can not create tracer: %w", err)
	}

	return NewTransactionRepositoryWithInterfaces(logger, client, tracer), nil
}

func NewTransactionRepositoryWithInterfaces(logger log.Logger, client gosoDynamodb.Client, tracer tracing.Tracer) *transactionRepository {
	return &transactionRepository{
		logger: logger,
		client: client,
		tracer: tracer,
	}
}

func (r transactionRepository) TransactGetItems(ctx context.Context, items []TransactGetItemBuilder) (*OperationResult, error) {
	res := newOperationResult(kindRead)

	if len(items) == 0 {
		return res, nil
	}

	_, span := r.tracer.StartSubSpan(ctx, "ddb.TransactGetItems")
	defer span.Finish()

	var err error
	transactionItems := make([]types.TransactGetItem, len(items))

	for i, v := range items {
		transactionItems[i], err = v.Build()
		if err != nil {
			return nil, err
		}
	}

	input := &dynamodb.TransactGetItemsInput{
		TransactItems:          transactionItems,
		ReturnConsumedCapacity: types.ReturnConsumedCapacityIndexes,
	}

	out, err := r.client.TransactGetItems(ctx, input)

	if exec.IsRequestCanceled(err) {
		return nil, exec.RequestCanceledError
	}

	if err != nil {
		return nil, gosoDynamodb.TransformTransactionError(err)
	}

	res.ConsumedCapacity.addSlice(out.ConsumedCapacity)

	for i, itemResponse := range out.Responses {
		err = UnmarshalMap(itemResponse.Item, items[i].GetItem())
		if err != nil {
			return nil, fmt.Errorf("could not unmarshal partial response: %w", err)
		}
	}

	return res, nil
}

func (r transactionRepository) TransactWriteItems(ctx context.Context, itemBuilders []TransactWriteItemBuilder) (*OperationResult, error) {
	return r.TransactWriteItemsIdempotent(ctx, itemBuilders, nil)
}

// TransactWriteItemsIdempotent
// ClientRequestToken enforces idempotency over a ten minute time frame
// https://docs.aws.amazon.com/amazondynamodb/latest/APIReference/API_TransactWriteItems.html#DDB-TransactWriteItems-request-ClientRequestToken
func (r transactionRepository) TransactWriteItemsIdempotent(ctx context.Context, itemBuilders []TransactWriteItemBuilder, clientRequestToken *string) (*OperationResult, error) {
	res := newOperationResult(kindMixed)

	if len(itemBuilders) == 0 {
		return res, nil
	}

	_, span := r.tracer.StartSubSpan(ctx, "ddb.TransactWriteItems")
	defer span.Finish()

	transactionItems := make([]types.TransactWriteItem, 0)
	for _, v := range itemBuilders {
		item, err := v.Build()
		if err != nil {
			return nil, err
		}

		transactionItems = append(transactionItems, *item)
	}

	input := &dynamodb.TransactWriteItemsInput{
		ClientRequestToken:     clientRequestToken,
		TransactItems:          transactionItems,
		ReturnConsumedCapacity: types.ReturnConsumedCapacityIndexes,
	}

	out, err := r.client.TransactWriteItems(ctx, input)

	if exec.IsRequestCanceled(err) {
		return nil, exec.RequestCanceledError
	}

	if err != nil {
		return nil, gosoDynamodb.TransformTransactionError(err)
	}

	res.ConsumedCapacity.addSlice(out.ConsumedCapacity)

	return res, gosoDynamodb.TransformTransactionError(err)
}
