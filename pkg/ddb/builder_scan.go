package ddb

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/expression"
	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
)

type ScanOperation struct {
	input      *dynamodb.ScanInput
	progress   *readProgress
	targetType interface{}
	result     *ScanResult
}

//go:generate mockery -name ScanBuilder
type ScanBuilder interface {
	WithIndex(name string) ScanBuilder
	WithFilter(filter expression.ConditionBuilder) ScanBuilder
	DisableTtlFilter() ScanBuilder
	WithProjection(projection interface{}) ScanBuilder
	WithLimit(limit int) ScanBuilder
	WithPageSize(size int) ScanBuilder
	WithSegment(segment int, total int) ScanBuilder
	Build(result interface{}) (*ScanOperation, error)
}

type scanBuilder struct {
	filterBuilder

	err          error
	metadata     *Metadata
	indexName    *string
	selected     FieldAware
	projection   interface{}
	limit        *int64
	pageSize     *int64
	segment      *int64
	segmentTotal *int64
}

func NewScanBuilder(metadata *Metadata) ScanBuilder {
	return &scanBuilder{
		metadata: metadata,
		selected: metadata.Main,
	}
}

func (b *scanBuilder) WithIndex(name string) ScanBuilder {
	index := b.metadata.Index(name)

	if index == nil {
		b.err = multierror.Append(b.err, fmt.Errorf("no index [%s] defined for table [%s]", name, b.metadata.TableName))
		return b
	}

	b.indexName = aws.String(name)
	b.selected = index

	return b
}

func (b *scanBuilder) WithFilter(filter expression.ConditionBuilder) ScanBuilder {
	b.filterCondition = &filter

	return b
}

func (b *scanBuilder) DisableTtlFilter() ScanBuilder {
	b.disableTtlFilter = true

	return b
}

func (b *scanBuilder) WithProjection(projection interface{}) ScanBuilder {
	b.projection = projection

	return b
}

func (b *scanBuilder) WithLimit(limit int) ScanBuilder {
	b.limit = aws.Int64(int64(limit))

	return b
}

func (b *scanBuilder) WithPageSize(size int) ScanBuilder {
	b.pageSize = aws.Int64(int64(size))

	return b
}

func (b *scanBuilder) WithSegment(segment int, total int) ScanBuilder {
	b.segment = aws.Int64(int64(segment))
	b.segmentTotal = aws.Int64(int64(total))

	return b
}

func (b *scanBuilder) Build(result interface{}) (*ScanOperation, error) {
	targetType := resolveTargetType(b.selected, b.projection, result)
	expr, err := b.buildExpression(targetType)

	if err != nil {
		return nil, err
	}

	progress := buildProgress(b.limit, b.pageSize)
	input := &dynamodb.ScanInput{
		TableName:                 aws.String(b.metadata.TableName),
		IndexName:                 b.indexName,
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
		FilterExpression:          expr.Filter(),
		ProjectionExpression:      expr.Projection(),
		Limit:                     b.limit,
		Segment:                   b.segment,
		TotalSegments:             b.segmentTotal,
	}

	operation := &ScanOperation{
		input:      input,
		progress:   progress,
		targetType: targetType,
		result:     newScanResult(),
	}

	return operation, nil
}

func (b *scanBuilder) buildExpression(result interface{}) (expression.Expression, error) {
	var err error
	var projectionExpr *expression.ProjectionBuilder

	parameters := 0
	exprBuilder := expression.NewBuilder()

	if filter := b.buildFilterCondition(b.metadata); filter != nil {
		exprBuilder = exprBuilder.WithFilter(*filter)
		parameters++
	}

	if projectionExpr, err = buildProjectionExpression(b.selected, result); err != nil {
		return expression.Expression{}, errors.Wrapf(err, "can not build projection for query")
	}

	if projectionExpr != nil {
		exprBuilder = exprBuilder.WithProjection(*projectionExpr)
		parameters++
	}

	if parameters == 0 {
		return expression.Expression{}, nil
	}

	return exprBuilder.Build()
}
