package ddb

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/expression"
	"github.com/hashicorp/go-multierror"
)

//go:generate mockery -name GetItemBuilder
type GetItemBuilder interface {
	WithHash(hashValue interface{}) GetItemBuilder
	WithRange(rangeValue interface{}) GetItemBuilder
	WithKeys(keys ...interface{}) GetItemBuilder
	WithProjection(rangeValue interface{}) GetItemBuilder
	Build(result interface{}) (*dynamodb.GetItemInput, error)
}

type getItemBuilder struct {
	err        error
	metadata   *Metadata
	keyBuilder keyBuilder
	projection interface{}
}

func NewGetItemBuilder(metadata *Metadata) GetItemBuilder {
	return &getItemBuilder{
		metadata: metadata,
		keyBuilder: keyBuilder{
			metadata: metadata.Main,
		},
	}
}

func (b *getItemBuilder) WithHash(hashValue interface{}) GetItemBuilder {
	b.keyBuilder.withHash(hashValue)

	return b
}

func (b *getItemBuilder) WithRange(rangeValue interface{}) GetItemBuilder {
	b.keyBuilder.withRange(rangeValue)

	return b
}

func (b *getItemBuilder) WithKeys(keys ...interface{}) GetItemBuilder {
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

func (b *getItemBuilder) WithProjection(projection interface{}) GetItemBuilder {
	b.projection = projection

	return b
}

func (b *getItemBuilder) Build(result interface{}) (*dynamodb.GetItemInput, error) {
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
		ExpressionAttributeNames: expr.Names(),
		ProjectionExpression:     expr.Projection(),
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
