package sqs

import (
	"context"
	"fmt"
)

const (
	AttributeSqsDelaySeconds           = "sqsDelaySeconds"
	AttributeSqsMessageGroupId         = "sqsMessageGroupId"
	AttributeSqsMessageDeduplicationId = "sqsMessageDeduplicationId"
	MaxDelaySeconds                    = 900
	DeduplicationIdMaxLen              = 128
	GroupIdMaxLen                      = 128
)

var (
	MessageDelaySeconds = Attribute{
		key: AttributeSqsDelaySeconds,
		convert: func(attribute any) (any, error) {
			delaySeconds, ok := attribute.(int32)

			if !ok {
				return nil, fmt.Errorf("failed to attach delay seconds: expected int32, got %v[%T]", attribute, attribute)
			}

			if delaySeconds > MaxDelaySeconds {
				return nil, fmt.Errorf("failed to attach delay seconds: delay of %d seconds is longer than maximum delay of %d seconds", delaySeconds, MaxDelaySeconds)
			}

			return delaySeconds, nil
		},
	}
	MessageDeduplicationId = Attribute{
		key: AttributeSqsMessageDeduplicationId,
		convert: func(attribute any) (any, error) {
			deduplicationId, ok := attribute.(string)

			if !ok {
				return nil, fmt.Errorf("failed to attach deduplication id: expected string, got %v[%T]", attribute, attribute)
			}

			if len(deduplicationId) > DeduplicationIdMaxLen {
				return nil, fmt.Errorf("failed to attach deduplication id: id %s of length %d is longer than maximum length of %d", deduplicationId, len(deduplicationId), DeduplicationIdMaxLen)
			}

			return deduplicationId, nil
		},
	}
	MessageGroupId = Attribute{
		key: AttributeSqsMessageGroupId,
		convert: func(attribute any) (any, error) {
			groupId, ok := attribute.(string)

			if !ok {
				return nil, fmt.Errorf("failed to attach group id: expected string, got %v[%T]", attribute, attribute)
			}

			if len(groupId) > GroupIdMaxLen {
				return nil, fmt.Errorf("failed to attach group id: id %s of length %d is longer than maximum length of %d", groupId, len(groupId), GroupIdMaxLen)
			}

			return groupId, nil
		},
	}
)

type Attribute struct {
	key     string
	convert func(attribute any) (any, error)
}

type AttributeProvider func(data any) (any, error)

type AttributeEncodeHandler struct {
	attribute         Attribute
	attributeProvider AttributeProvider
}

func NewAttributeEncodeHandler(attribute Attribute, attributeProvider AttributeProvider) *AttributeEncodeHandler {
	return &AttributeEncodeHandler{
		attribute:         attribute,
		attributeProvider: attributeProvider,
	}
}

func (g *AttributeEncodeHandler) Encode(ctx context.Context, data any, attributes map[string]any) (context.Context, map[string]any, error) {
	attribute, err := g.attributeProvider(data)
	if err != nil {
		return ctx, attributes, err
	}

	if attribute != nil {
		value, err := g.attribute.convert(attribute)
		if err != nil {
			return ctx, attributes, err
		}

		attributes[g.attribute.key] = value
	}

	return ctx, attributes, nil
}

func (g *AttributeEncodeHandler) Decode(ctx context.Context, _ any, attributes map[string]any) (context.Context, map[string]any, error) {
	return ctx, attributes, nil
}
