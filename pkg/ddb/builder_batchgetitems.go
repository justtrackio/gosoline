package ddb

import (
	"fmt"

	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/expression"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/hashicorp/go-multierror"
	"github.com/justtrackio/gosoline/pkg/clock"
	"github.com/justtrackio/gosoline/pkg/refl"
)

//go:generate go run github.com/vektra/mockery/v2 --name BatchGetItemsBuilder
type BatchGetItemsBuilder interface {
	WithKeys(values ...any) BatchGetItemsBuilder
	WithKeyPairs(pairs [][]any) BatchGetItemsBuilder
	WithHashKeys(hashKeys any) BatchGetItemsBuilder
	DisableTtlFilter() BatchGetItemsBuilder
	WithProjection(projection any) BatchGetItemsBuilder
	WithConsistentRead(consistentRead bool) BatchGetItemsBuilder
	Build(result any) (*dynamodb.BatchGetItemInput, error)
}

type batchGetItemsBuilder struct {
	filterBuilder

	err            error
	keyBuilder     keyBuilder
	keyPairs       [][]any
	consistentRead *bool
	projection     any
}

func NewBatchGetItemsBuilder(metadata *Metadata, clock clock.Clock) BatchGetItemsBuilder {
	return &batchGetItemsBuilder{
		filterBuilder: newFilterBuilder(metadata, clock),

		keyBuilder: keyBuilder{
			metadata: metadata.Main,
		},
		keyPairs: make([][]any, 0, 100),
	}
}

func (b *batchGetItemsBuilder) WithKeys(values ...any) BatchGetItemsBuilder {
	b.keyPairs = append(b.keyPairs, values)

	return b
}

func (b *batchGetItemsBuilder) WithKeyPairs(pairs [][]any) BatchGetItemsBuilder {
	b.keyPairs = append(b.keyPairs, pairs...)

	return b
}

func (b *batchGetItemsBuilder) WithHashKeys(hashKeys any) BatchGetItemsBuilder {
	slice, err := refl.InterfaceToInterfaceSlice(hashKeys)
	if err != nil {
		b.err = multierror.Append(b.err, err)
	}

	for _, hash := range slice {
		b.WithKeys(hash)
	}

	return b
}

func (b *batchGetItemsBuilder) DisableTtlFilter() BatchGetItemsBuilder {
	b.disableTtlFilter = true

	return b
}

func (b *batchGetItemsBuilder) WithProjection(projection any) BatchGetItemsBuilder {
	b.projection = projection

	return b
}

func (b *batchGetItemsBuilder) WithConsistentRead(consistentRead bool) BatchGetItemsBuilder {
	b.consistentRead = &consistentRead

	return b
}

func (b *batchGetItemsBuilder) Build(result any) (*dynamodb.BatchGetItemInput, error) {
	if b.projection == nil {
		b.projection = result
	}

	if b.err != nil {
		return nil, b.err
	}

	if len(b.keyPairs) == 0 {
		return nil, fmt.Errorf("no key pairs provided to select items")
	}

	keyAttributes := make([]map[string]types.AttributeValue, len(b.keyPairs))

	for i, keys := range b.keyPairs {
		attributeValues, err := b.keyBuilder.fromValues(keys...)
		if err != nil {
			return nil, err
		}

		keyAttributes[i] = attributeValues
	}

	expr, err := b.buildExpression()
	if err != nil {
		return nil, err
	}

	input := &dynamodb.BatchGetItemInput{
		RequestItems: map[string]types.KeysAndAttributes{
			b.metadata.TableName: {
				Keys:                     keyAttributes,
				ConsistentRead:           b.consistentRead,
				ExpressionAttributeNames: expr.Names(),
				ProjectionExpression:     expr.Projection(),
			},
		},
		ReturnConsumedCapacity: types.ReturnConsumedCapacityIndexes,
	}

	return input, nil
}

func (b *batchGetItemsBuilder) buildExpression() (expression.Expression, error) {
	projection, err := buildProjectionExpression(b.metadata.Main, b.projection)
	if err != nil {
		return expression.Expression{}, err
	}

	if projection == nil {
		return expression.Expression{}, nil
	}

	return expression.NewBuilder().WithProjection(*projection).Build()
}
