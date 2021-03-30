package ddb

import (
	"fmt"
	"github.com/applike/gosoline/pkg/clock"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/expression"
	"github.com/hashicorp/go-multierror"
)

const (
	CompBetween    = "between"
	CompBeginsWith = "beginsWith"
	CompEq         = "="
	CompGt         = ">"
	CompGte        = ">="
	CompLt         = "<"
	CompLte        = "<="
)

type QueryOperation struct {
	input      *dynamodb.QueryInput
	iterator   *pageIterator
	targetType interface{}
	result     *QueryResult
}

type keyExprBuilder func() expression.KeyConditionBuilder

//go:generate mockery -name QueryBuilder
type QueryBuilder interface {
	WithIndex(name string) QueryBuilder
	WithHash(value interface{}) QueryBuilder
	WithRange(comp string, values ...interface{}) QueryBuilder
	WithRangeBetween(lower interface{}, upper interface{}) QueryBuilder
	WithRangeBeginsWith(prefix string) QueryBuilder
	WithRangeEq(value interface{}) QueryBuilder
	WithRangeGt(value interface{}) QueryBuilder
	WithRangeGte(value interface{}) QueryBuilder
	WithRangeLt(value interface{}) QueryBuilder
	WithRangeLte(value interface{}) QueryBuilder
	WithFilter(filter expression.ConditionBuilder) QueryBuilder
	DisableTtlFilter() QueryBuilder
	WithProjection(projection interface{}) QueryBuilder
	WithLimit(limit int) QueryBuilder
	WithPageSize(size int) QueryBuilder
	WithDescendingOrder() QueryBuilder
	WithConsistentRead(consistentRead bool) QueryBuilder
	Build(result interface{}) (*QueryOperation, error)
}

type queryBuilder struct {
	filterBuilder

	indexName *string
	selected  FieldAware
	err       error

	hashExprBuilder  keyExprBuilder
	rangeExprBuilder keyExprBuilder
	projection       interface{}
	limit            *int64
	pageSize         *int64
	scanIndexForward *bool
	consistentRead   *bool
}

func NewQueryBuilder(metadata *Metadata, clock clock.Clock) QueryBuilder {
	return &queryBuilder{
		filterBuilder: newFilterBuilder(metadata, clock),

		selected: metadata.Main,
	}
}

func (b *queryBuilder) WithIndex(name string) QueryBuilder {
	index := b.metadata.Index(name)

	if index == nil {
		b.err = multierror.Append(b.err, fmt.Errorf("no index [%s] defined for table [%s]", name, b.metadata.TableName))
		return b
	}

	b.indexName = aws.String(name)
	b.selected = index

	return b
}

func (b *queryBuilder) WithHash(value interface{}) QueryBuilder {
	b.hashExprBuilder = func() expression.KeyConditionBuilder {
		return expression.KeyEqual(expression.Key(*b.selected.GetHashKey()), expression.Value(value))
	}

	return b
}

func (b *queryBuilder) WithRange(comp string, values ...interface{}) QueryBuilder {
	if len(values) == 0 {
		b.err = multierror.Append(b.err, fmt.Errorf("there are no values for the range query on table %s", b.metadata.TableName))
		return b
	}

	switch comp {
	case CompBetween:
		if len(values) != 2 {
			b.err = multierror.Append(b.err, fmt.Errorf("there are 2 values required for a range between query on table %s", b.metadata.TableName))
			return b
		}

		return b.WithRangeBetween(values[0], values[1])

	case CompBeginsWith:
		prefix, ok := values[0].(string)

		if !ok {
			b.err = multierror.Append(b.err, fmt.Errorf("paramter for a range beginsWith query has to be of type string on table %s", b.metadata.TableName))
			return b
		}

		return b.WithRangeBeginsWith(prefix)

	case CompEq:
		return b.WithRangeEq(values[0])

	case CompGt:
		return b.WithRangeGt(values[0])

	case CompGte:
		return b.WithRangeGte(values[0])

	case CompLt:
		return b.WithRangeLt(values[0])

	case CompLte:
		return b.WithRangeLte(values[0])
	}

	b.err = multierror.Append(b.err, fmt.Errorf("unkown query operation [%s] on table %s", comp, b.metadata.TableName))

	return b
}

func (b *queryBuilder) WithRangeBetween(lower interface{}, upper interface{}) QueryBuilder {
	b.rangeExprBuilder = func() expression.KeyConditionBuilder {
		return expression.KeyBetween(expression.Key(*b.selected.GetRangeKey()), expression.Value(lower), expression.Value(upper))
	}

	return b
}

func (b *queryBuilder) WithRangeBeginsWith(prefix string) QueryBuilder {
	b.rangeExprBuilder = func() expression.KeyConditionBuilder {
		return expression.KeyBeginsWith(expression.Key(*b.selected.GetRangeKey()), prefix)
	}

	return b
}

func (b *queryBuilder) WithRangeEq(value interface{}) QueryBuilder {
	b.rangeExprBuilder = func() expression.KeyConditionBuilder {
		return expression.KeyEqual(expression.Key(*b.selected.GetRangeKey()), expression.Value(value))
	}

	return b
}

func (b *queryBuilder) WithRangeGt(value interface{}) QueryBuilder {
	b.rangeExprBuilder = func() expression.KeyConditionBuilder {
		return expression.KeyGreaterThan(expression.Key(*b.selected.GetRangeKey()), expression.Value(value))
	}

	return b
}

func (b *queryBuilder) WithRangeGte(value interface{}) QueryBuilder {
	b.rangeExprBuilder = func() expression.KeyConditionBuilder {
		return expression.KeyGreaterThanEqual(expression.Key(*b.selected.GetRangeKey()), expression.Value(value))
	}

	return b
}

func (b *queryBuilder) WithRangeLt(value interface{}) QueryBuilder {
	b.rangeExprBuilder = func() expression.KeyConditionBuilder {
		return expression.KeyLessThan(expression.Key(*b.selected.GetRangeKey()), expression.Value(value))
	}

	return b
}

func (b *queryBuilder) WithRangeLte(value interface{}) QueryBuilder {
	b.rangeExprBuilder = func() expression.KeyConditionBuilder {
		return expression.KeyLessThanEqual(expression.Key(*b.selected.GetRangeKey()), expression.Value(value))
	}

	return b
}

func (b *queryBuilder) WithFilter(filter expression.ConditionBuilder) QueryBuilder {
	b.filterCondition = &filter

	return b
}

func (b *queryBuilder) DisableTtlFilter() QueryBuilder {
	b.disableTtlFilter = true

	return b
}

func (b *queryBuilder) WithProjection(projection interface{}) QueryBuilder {
	b.projection = projection

	return b
}

func (b *queryBuilder) WithLimit(limit int) QueryBuilder {
	b.limit = aws.Int64(int64(limit))

	return b
}

func (b *queryBuilder) WithPageSize(size int) QueryBuilder {
	b.pageSize = aws.Int64(int64(size))

	return b
}

func (b *queryBuilder) WithDescendingOrder() QueryBuilder {
	b.scanIndexForward = aws.Bool(false)

	return b
}

func (b *queryBuilder) WithConsistentRead(consistentRead bool) QueryBuilder {
	b.consistentRead = &consistentRead

	return b
}

func (b *queryBuilder) Build(result interface{}) (*QueryOperation, error) {
	var err error
	var keyCondition expression.KeyConditionBuilder
	var projectionExpr *expression.ProjectionBuilder

	if b.err != nil {
		return nil, b.err
	}

	exprBuilder := expression.NewBuilder()

	if keyCondition, err = b.buildKeyCondition(); err != nil {
		return nil, err
	}

	exprBuilder = exprBuilder.WithKeyCondition(keyCondition)

	if filter := b.buildFilterCondition(); filter != nil {
		exprBuilder = exprBuilder.WithFilter(*filter)
	}

	targetType := resolveTargetType(b.selected, b.projection, result)

	if projectionExpr, err = buildProjectionExpression(b.selected, targetType); err != nil {
		return nil, fmt.Errorf("can not build projection for query: %w", err)
	}

	if projectionExpr != nil {
		exprBuilder = exprBuilder.WithProjection(*projectionExpr)
	}

	expr, err := exprBuilder.Build()

	if err != nil {
		return nil, err
	}

	progress := buildPageIterator(b.limit, b.pageSize)
	input := &dynamodb.QueryInput{
		TableName:                 aws.String(b.metadata.TableName),
		IndexName:                 b.indexName,
		ConsistentRead:            b.consistentRead,
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
		KeyConditionExpression:    expr.KeyCondition(),
		FilterExpression:          expr.Filter(),
		ProjectionExpression:      expr.Projection(),
		Limit:                     progress.size,
		ScanIndexForward:          b.scanIndexForward,
	}

	operation := &QueryOperation{
		input:      input,
		iterator:   progress,
		targetType: targetType,
		result:     newQueryResult(),
	}

	return operation, nil
}

func (b *queryBuilder) buildKeyCondition() (expression.KeyConditionBuilder, error) {
	if b.selected.GetHashKey() == nil {
		return expression.KeyConditionBuilder{}, fmt.Errorf("no hash key defined for table %s", b.metadata.TableName)
	}

	if b.hashExprBuilder == nil {
		return expression.KeyConditionBuilder{}, fmt.Errorf("no value for the hash key provided for table %s", b.metadata.TableName)
	}

	condition := b.hashExprBuilder()

	if b.rangeExprBuilder != nil {
		if b.selected.GetRangeKey() == nil {
			return expression.KeyConditionBuilder{}, fmt.Errorf("no range key defined for table %s", b.metadata.TableName)
		}

		rangeCondition := b.rangeExprBuilder()
		condition = condition.And(rangeCondition)
	}

	return condition, nil
}
