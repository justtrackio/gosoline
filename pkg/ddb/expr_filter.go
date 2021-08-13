package ddb

import (
	"github.com/applike/gosoline/pkg/clock"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/expression"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

type ttlStruct struct {
	Ttl int64 `json:"ttl"`
}

type ttlFilterer interface {
	PerformFilterCondition(item map[string]types.AttributeValue) (bool, error)
}

type filterBuilder struct {
	metadata         *Metadata
	filterCondition  *expression.ConditionBuilder
	disableTtlFilter bool
	clock            clock.Clock
}

func newFilterBuilder(metadata *Metadata, clock clock.Clock) filterBuilder {
	return filterBuilder{
		metadata: metadata,
		clock:    clock,
	}
}

func (b *filterBuilder) buildFilterCondition() *expression.ConditionBuilder {
	ttl := b.metadata.TimeToLive

	if !ttl.Enabled || b.disableTtlFilter {
		return b.filterCondition
	}

	now := b.clock.Now().Unix()
	expr := expression.GreaterThan(expression.Name(ttl.Field), expression.Value(now))

	if b.filterCondition == nil {
		return &expr
	}

	expr = b.filterCondition.And(expr)

	return &expr
}

func (b *filterBuilder) PerformFilterCondition(item map[string]types.AttributeValue) (bool, error) {
	ttl := b.metadata.TimeToLive

	if !ttl.Enabled || b.disableTtlFilter {
		return true, nil
	}

	now := b.clock.Now().Unix()

	s := &ttlStruct{}
	err := UnmarshalMap(map[string]types.AttributeValue{
		"ttl": item[ttl.Field],
	}, s)
	if err != nil {
		return false, err
	}

	return s.Ttl > now, nil
}
