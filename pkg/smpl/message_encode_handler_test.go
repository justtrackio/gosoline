package smpl_test

import (
	"context"
	"testing"

	"github.com/justtrackio/gosoline/pkg/smpl"
	"github.com/justtrackio/gosoline/pkg/smpl/smplctx"
	"github.com/stretchr/testify/suite"
)

type MessageWithSamplingEncoderTestSuite struct {
	suite.Suite
}

func TestMessageWithSamplingEncoderTestSuite(t *testing.T) {
	suite.Run(t, new(MessageWithSamplingEncoderTestSuite))
}

func (s *MessageWithSamplingEncoderTestSuite) TestEncode_SampledTrue() {
	encoder := smpl.NewMessageWithSamplingEncoder()
	ctx := smplctx.WithSampling(context.Background(), smplctx.Sampling{Sampled: true})
	attributes := make(map[string]string)

	newCtx, newAttrs, err := encoder.Encode(ctx, nil, attributes)

	s.NoError(err)
	s.Equal(ctx, newCtx, "Context should be returned unchanged")
	s.Equal("true", newAttrs["sampled"], "Should encode sampled=true as 'true'")
}

func (s *MessageWithSamplingEncoderTestSuite) TestEncode_SampledFalse() {
	encoder := smpl.NewMessageWithSamplingEncoder()
	ctx := smplctx.WithSampling(context.Background(), smplctx.Sampling{Sampled: false})
	attributes := make(map[string]string)

	newCtx, newAttrs, err := encoder.Encode(ctx, nil, attributes)

	s.NoError(err)
	s.Equal(ctx, newCtx, "Context should be returned unchanged")
	s.Equal("false", newAttrs["sampled"], "Should encode sampled=false as 'false'")
}

func (s *MessageWithSamplingEncoderTestSuite) TestEncode_NoSamplingInContext() {
	encoder := smpl.NewMessageWithSamplingEncoder()
	ctx := context.Background()
	attributes := make(map[string]string)

	newCtx, newAttrs, err := encoder.Encode(ctx, nil, attributes)

	s.NoError(err)
	s.Equal(ctx, newCtx, "Context should be returned unchanged")
	s.Equal("true", newAttrs["sampled"], "Should default to sampled=true when no sampling in context")
}

func (s *MessageWithSamplingEncoderTestSuite) TestEncode_PreservesExistingAttributes() {
	encoder := smpl.NewMessageWithSamplingEncoder()
	ctx := smplctx.WithSampling(context.Background(), smplctx.Sampling{Sampled: true})
	attributes := map[string]string{
		"existing-key": "existing-value",
	}

	newCtx, newAttrs, err := encoder.Encode(ctx, nil, attributes)

	s.NoError(err)
	s.Equal(ctx, newCtx, "Context should be returned unchanged")
	s.Equal("true", newAttrs["sampled"], "Should encode sampled attribute")
	s.Equal("existing-value", newAttrs["existing-key"], "Should preserve existing attributes")
}

func (s *MessageWithSamplingEncoderTestSuite) TestDecode_ValidTrue() {
	encoder := smpl.NewMessageWithSamplingEncoder()
	ctx := context.Background()
	attributes := map[string]string{
		"sampled": "true",
	}

	newCtx, newAttrs, err := encoder.Decode(ctx, nil, attributes)

	s.NoError(err)
	s.True(smplctx.IsSampled(newCtx), "Context should be sampled=true")
	s.NotContains(newAttrs, "sampled", "Should remove 'sampled' attribute after decoding")
}

func (s *MessageWithSamplingEncoderTestSuite) TestDecode_ValidFalse() {
	encoder := smpl.NewMessageWithSamplingEncoder()
	ctx := context.Background()
	attributes := map[string]string{
		"sampled": "false",
	}

	newCtx, newAttrs, err := encoder.Decode(ctx, nil, attributes)

	s.NoError(err)
	s.False(smplctx.IsSampled(newCtx), "Context should be sampled=false")
	s.NotContains(newAttrs, "sampled", "Should remove 'sampled' attribute after decoding")
}

func (s *MessageWithSamplingEncoderTestSuite) TestDecode_Numeric() {
	encoder := smpl.NewMessageWithSamplingEncoder()
	ctx := context.Background()

	// Test with "1" (should parse as true)
	attributes1 := map[string]string{
		"sampled": "1",
	}
	newCtx1, newAttrs1, err1 := encoder.Decode(ctx, nil, attributes1)
	s.NoError(err1)
	s.True(smplctx.IsSampled(newCtx1), "Should parse '1' as true")
	s.NotContains(newAttrs1, "sampled")

	// Test with "0" (should parse as false)
	attributes0 := map[string]string{
		"sampled": "0",
	}
	newCtx0, newAttrs0, err0 := encoder.Decode(ctx, nil, attributes0)
	s.NoError(err0)
	s.False(smplctx.IsSampled(newCtx0), "Should parse '0' as false")
	s.NotContains(newAttrs0, "sampled")
}

func (s *MessageWithSamplingEncoderTestSuite) TestDecode_MissingAttribute() {
	encoder := smpl.NewMessageWithSamplingEncoder()
	ctx := context.Background()
	attributes := map[string]string{
		"other-key": "other-value",
	}

	newCtx, newAttrs, err := encoder.Decode(ctx, nil, attributes)

	s.NoError(err)
	s.Equal(ctx, newCtx, "Context should be returned unchanged when attribute is missing")
	s.Equal(attributes, newAttrs, "Attributes should be unchanged when 'sampled' is missing")
}

func (s *MessageWithSamplingEncoderTestSuite) TestDecode_InvalidBoolValue() {
	encoder := smpl.NewMessageWithSamplingEncoder()
	ctx := context.Background()
	attributes := map[string]string{
		"sampled": "not-a-bool",
	}

	newCtx, newAttrs, err := encoder.Decode(ctx, nil, attributes)

	s.Error(err)
	s.Contains(err.Error(), "failed to parse sampled attribute")
	s.Equal(ctx, newCtx, "Context should be returned unchanged on error")
	s.Equal(attributes, newAttrs, "Attributes should be returned unchanged on error")
}

func (s *MessageWithSamplingEncoderTestSuite) TestDecode_PreservesOtherAttributes() {
	encoder := smpl.NewMessageWithSamplingEncoder()
	ctx := context.Background()
	attributes := map[string]string{
		"sampled":      "true",
		"existing-key": "existing-value",
	}

	newCtx, newAttrs, err := encoder.Decode(ctx, nil, attributes)

	s.NoError(err)
	s.True(smplctx.IsSampled(newCtx), "Context should be sampled=true")
	s.NotContains(newAttrs, "sampled", "Should remove 'sampled' attribute after decoding")
	s.Equal("existing-value", newAttrs["existing-key"], "Should preserve other attributes")
}

func (s *MessageWithSamplingEncoderTestSuite) TestDecode_EmptyString() {
	encoder := smpl.NewMessageWithSamplingEncoder()
	ctx := context.Background()
	attributes := map[string]string{
		"sampled": "",
	}

	newCtx, newAttrs, err := encoder.Decode(ctx, nil, attributes)

	s.Error(err)
	s.Contains(err.Error(), "failed to parse sampled attribute")
	s.Equal(ctx, newCtx, "Context should be returned unchanged on error")
	s.Equal(attributes, newAttrs, "Attributes should be returned unchanged on error")
}

func (s *MessageWithSamplingEncoderTestSuite) TestRoundTrip() {
	encoder := smpl.NewMessageWithSamplingEncoder()

	// Test round-trip with sampled=true
	ctx1 := smplctx.WithSampling(context.Background(), smplctx.Sampling{Sampled: true})
	attrs1 := make(map[string]string)

	_, encodedAttrs1, err := encoder.Encode(ctx1, nil, attrs1)
	s.NoError(err)

	decodedCtx1, _, err := encoder.Decode(context.Background(), nil, encodedAttrs1)
	s.NoError(err)
	s.True(smplctx.IsSampled(decodedCtx1), "Round-trip should preserve sampled=true")

	// Test round-trip with sampled=false
	ctx2 := smplctx.WithSampling(context.Background(), smplctx.Sampling{Sampled: false})
	attrs2 := make(map[string]string)

	_, encodedAttrs2, err := encoder.Encode(ctx2, nil, attrs2)
	s.NoError(err)

	decodedCtx2, _, err := encoder.Decode(context.Background(), nil, encodedAttrs2)
	s.NoError(err)
	s.False(smplctx.IsSampled(decodedCtx2), "Round-trip should preserve sampled=false")
}
