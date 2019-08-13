package ddb

import (
	"github.com/aws/aws-sdk-go/service/dynamodb/expression"
	"time"
)

type filterBuilder struct {
	filterCondition  *expression.ConditionBuilder
	disableTtlFilter bool
}

func (b *filterBuilder) buildFilterCondition(metadata *Metadata) *expression.ConditionBuilder {
	ttl := metadata.TimeToLive

	if !ttl.Enabled || b.disableTtlFilter {
		return b.filterCondition
	}

	now := time.Now().Unix()
	expr := expression.GreaterThan(expression.Name(ttl.Field), expression.Value(now))

	if b.filterCondition == nil {
		return &expr
	}

	expr = b.filterCondition.And(expr)

	return &expr
}
