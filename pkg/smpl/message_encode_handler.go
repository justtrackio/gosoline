package smpl

import (
	"context"
	"fmt"
	"strconv"

	"github.com/justtrackio/gosoline/pkg/smpl/smplctx"
)

// MessageWithSamplingEncoder encodes/decodes the sampling decision into message attributes.
// It uses the attribute "sampled" to propagate the decision across async boundaries.
type MessageWithSamplingEncoder struct{}

// NewMessageWithSamplingEncoder creates a new MessageWithSamplingEncoder.
func NewMessageWithSamplingEncoder() *MessageWithSamplingEncoder {
	return &MessageWithSamplingEncoder{}
}

// Encode writes the current sampling decision from the context into the "sampled" attribute.
func (m MessageWithSamplingEncoder) Encode(ctx context.Context, _ any, attributes map[string]string) (context.Context, map[string]string, error) {
	sampling := smplctx.GetSampling(ctx)
	attributes["sampled"] = strconv.FormatBool(sampling.Sampled)

	return ctx, attributes, nil
}

// Decode reads the "sampled" attribute and updates the context with the propagated decision.
func (m MessageWithSamplingEncoder) Decode(ctx context.Context, _ any, attributes map[string]string) (context.Context, map[string]string, error) {
	var ok, sampled bool
	var err error

	if _, ok = attributes["sampled"]; !ok {
		return ctx, attributes, nil
	}

	if sampled, err = strconv.ParseBool(attributes["sampled"]); err != nil {
		return ctx, attributes, fmt.Errorf("failed to parse sampled attribute: %w", err)
	}
	ctx = smplctx.WithSampling(ctx, smplctx.Sampling{
		Sampled: sampled,
	})
	delete(attributes, "sampled")

	return ctx, attributes, nil
}
