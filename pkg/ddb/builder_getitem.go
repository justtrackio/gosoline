package ddb

import (
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/expression"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/hashicorp/go-multierror"
	"github.com/justtrackio/gosoline/pkg/clock"
)

//go:generate go run github.com/vektra/mockery/v2 --name GetItemBuilder
type GetItemBuilder interface {
	WithHash(hashValue any) GetItemBuilder
	WithRange(rangeValue any) GetItemBuilder
	WithKeys(keys ...any) GetItemBuilder
	DisableTtlFilter() GetItemBuilder
	WithProjection(rangeValue any) GetItemBuilder
	WithConsistentRead(consistentRead bool) GetItemBuilder
	Build(result any) (*dynamodb.GetItemInput, error)
}

type getItemBuilder struct {
	filterBuilder

	err            error
	keyBuilder     keyBuilder
	consistentRead *bool
	projection     any
}

func NewGetItemBuilder(metadata *Metadata, clock clock.Clock) GetItemBuilder {
	return &getItemBuilder{
		filterBuilder: newFilterBuilder(metadata, clock),

		keyBuilder: keyBuilder{
			metadata: metadata.Main,
		},
	}
}

func (b *getItemBuilder) WithHash(hashValue any) GetItemBuilder {
	b.keyBuilder.withHash(hashValue)

	return b
}

func (b *getItemBuilder) WithRange(rangeValue any) GetItemBuilder {
	b.keyBuilder.withRange(rangeValue)

	return b
}

func (b *getItemBuilder) WithKeys(keys ...any) GetItemBuilder {
	if len(keys) == 0 {
		return b
	}

	b.WithHash(keys[0])

	if len(keys) > 2 {
		b.err = multierror.Append(b.err, fmt.Errorf("more than two keys provided for WithKeys"))

		return b
	}

	b.WithRange(keys[1])

	return b
}

func (b *getItemBuilder) DisableTtlFilter() GetItemBuilder {
	b.disableTtlFilter = true

	return b
}

func (b *getItemBuilder) WithProjection(projection any) GetItemBuilder {
	b.projection = projection

	return b
}

func (b *getItemBuilder) WithConsistentRead(consistentRead bool) GetItemBuilder {
	b.consistentRead = &consistentRead

	return b
}

func (b *getItemBuilder) Build(result any) (*dynamodb.GetItemInput, error) {
	if b.err != nil {
		return nil, b.err
	}

	if b.projection == nil {
		b.projection = result
	}

	keys, err := b.keyBuilder.buildKey(result)
	if err != nil {
		return nil, err
	}

	expr, err := b.buildExpression()
	if err != nil {
		return nil, err
	}

	input := &dynamodb.GetItemInput{
		TableName:                aws.String(b.metadata.TableName),
		Key:                      keys,
		ConsistentRead:           b.consistentRead,
		ExpressionAttributeNames: expr.Names(),
		ProjectionExpression:     expr.Projection(),
		ReturnConsumedCapacity:   types.ReturnConsumedCapacityIndexes,
	}

	return input, nil
}

func (b *getItemBuilder) buildExpression() (expression.Expression, error) {
	projection, err := buildProjectionExpression(b.metadata.Main, b.projection)
	if err != nil {
		return expression.Expression{}, err
	}

	if projection == nil {
		return expression.Expression{}, nil
	}

	return expression.NewBuilder().WithProjection(*projection).Build()
}
