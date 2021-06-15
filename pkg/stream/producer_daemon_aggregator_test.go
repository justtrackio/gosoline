package stream_test

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"github.com/applike/gosoline/pkg/encoding/base64"
	"github.com/applike/gosoline/pkg/encoding/json"
	"github.com/applike/gosoline/pkg/stream"
	"github.com/stretchr/testify/assert"
	"math/rand"
	"strings"
	"testing"
)

type aggregatorTestCase struct {
	aggregationSize    int
	aggregationMaxSize int
	compression        stream.CompressionType

	messages []*stream.Message
	flushes  []expectedFlush
}

func (tc aggregatorTestCase) run(t *testing.T) {
	agg, err := stream.NewProducerDaemonAggregator(stream.ProducerDaemonSettings{
		AggregationSize:    tc.aggregationSize,
		AggregationMaxSize: tc.aggregationMaxSize,
	}, tc.compression)
	assert.NoError(t, err)

	flushes := make([]stream.AggregateFlush, 0)

	for _, msg := range tc.messages {
		flushList, err := agg.Write(msg)
		assert.NoError(t, err)

		flushes = append(flushes, flushList...)
	}

	flush, err := agg.Flush()
	assert.NoError(t, err)
	if flush.MessageCount > 0 {
		flushes = append(flushes, *flush)
	}

	assert.Len(t, flushes, len(tc.flushes))

	expectedFlushes := make([]stream.AggregateFlush, len(tc.flushes))
	for i, expectedFlush := range tc.flushes {
		expectedFlushes[i] = expectedFlush.encode(tc.compression)

		if expectedFlush.validate != nil && len(flushes) > i {
			err := expectedFlush.validate(&flushes[i])
			assert.NoError(t, err)
		}
	}

	assert.Equal(t, expectedFlushes, flushes)
}

type expectedFlush struct {
	messages []*stream.Message
	validate func(flush *stream.AggregateFlush) error
}

func (f expectedFlush) encode(compression stream.CompressionType) stream.AggregateFlush {
	body, err := json.Marshal(f.messages)
	if err != nil {
		panic(err)
	}

	attributes := map[string]interface{}{}
	if compression == stream.CompressionGZip {
		attributes[stream.AttributeCompression] = compression

		var buffer bytes.Buffer
		writer := gzip.NewWriter(&buffer)

		if _, err := writer.Write(body); err != nil {
			panic(err)
		}

		if err := writer.Close(); err != nil {
			panic(err)
		}

		body = base64.Encode(buffer.Bytes())
	}

	return stream.AggregateFlush{
		Attributes:   attributes,
		Body:         string(body),
		MessageCount: len(f.messages),
	}
}

func mkTestMessage(t *testing.T, body interface{}, attributes map[string]interface{}) *stream.Message {
	attributes[stream.AttributeEncoding] = stream.EncodingJson

	bodyBytes, err := json.Marshal(body)
	assert.NoError(t, err)

	return &stream.Message{
		Attributes: attributes,
		Body:       string(bodyBytes),
	}
}

func TestProducerDaemonAggregator_CountRestricted(t *testing.T) {
	messages := []*stream.Message{
		mkTestMessage(t, "message 1", map[string]interface{}{}),
		mkTestMessage(t, "message 2", map[string]interface{}{}),
		mkTestMessage(t, "message 3", map[string]interface{}{
			"attribute": "value",
		}),
		mkTestMessage(t, "message 4", map[string]interface{}{}),
		mkTestMessage(t, "message 5", map[string]interface{}{}),
		mkTestMessage(t, "message 6", map[string]interface{}{
			"attribute": "another value",
		}),
		mkTestMessage(t, "message 7", map[string]interface{}{}),
	}

	aggregatorTestCase{
		aggregationSize:    5,
		aggregationMaxSize: 0,
		compression:        stream.CompressionNone,
		messages:           messages,
		flushes: []expectedFlush{
			{
				messages: messages[0:5],
			},
			{
				messages: messages[5:],
			},
		},
	}.run(t)
}

func TestProducerDaemonAggregator_SizeRestricted(t *testing.T) {
	messages := []*stream.Message{
		mkTestMessage(t, strings.Repeat("1", 50), map[string]interface{}{}),
		mkTestMessage(t, strings.Repeat("2", 50), map[string]interface{}{}),
		mkTestMessage(t, strings.Repeat("3", 50), map[string]interface{}{
			"attribute": fmt.Sprintf("l%sng value", strings.Repeat("o", 100)),
		}),
		mkTestMessage(t, strings.Repeat("4", 50), map[string]interface{}{}),
		mkTestMessage(t, strings.Repeat("5", 50), map[string]interface{}{}),
		mkTestMessage(t, strings.Repeat("6", 50), map[string]interface{}{
			"attribute": "another value",
		}),
		mkTestMessage(t, strings.Repeat("7", 50), map[string]interface{}{}),
	}

	belowSizeLimit := getAggregatorValidateFunction(5_000)

	aggregatorTestCase{
		aggregationSize:    5_000,
		aggregationMaxSize: 300,
		compression:        stream.CompressionNone,
		messages:           messages,
		flushes: []expectedFlush{
			{
				messages: messages[0:2],
				validate: belowSizeLimit,
			},
			{
				messages: messages[2:3],
				validate: belowSizeLimit,
			},
			{
				messages: messages[3:5],
				validate: belowSizeLimit,
			},
			{
				messages: messages[5:],
				validate: belowSizeLimit,
			},
		},
	}.run(t)
}

func TestProducerDaemonAggregator_CompressedSizeRestrictedSmall(t *testing.T) {
	messages := []*stream.Message{
		mkTestMessage(t, strings.Repeat("1", 5_000), map[string]interface{}{}),
		mkTestMessage(t, strings.Repeat("2", 5_000), map[string]interface{}{}),
		mkTestMessage(t, strings.Repeat("3", 5_000), map[string]interface{}{
			"attribute": fmt.Sprintf("l%sng value", strings.Repeat("o", 100)),
		}),
		mkTestMessage(t, strings.Repeat("4", 5_000), map[string]interface{}{}),
		mkTestMessage(t, strings.Repeat("5", 5_000), map[string]interface{}{}),
		mkTestMessage(t, strings.Repeat("6", 5_000), map[string]interface{}{
			"attribute": "another value",
		}),
		mkTestMessage(t, strings.Repeat("7", 5_000), map[string]interface{}{}),
	}

	belowSizeLimit := getAggregatorValidateFunction(5_000)

	aggregatorTestCase{
		aggregationSize:    5_000,
		aggregationMaxSize: 5_000,
		compression:        stream.CompressionGZip,
		messages:           messages,
		flushes: []expectedFlush{
			{
				messages: messages[0:1],
				validate: belowSizeLimit,
			},
			{
				messages: messages[1:],
				validate: belowSizeLimit,
			},
		},
	}.run(t)
}

func TestProducerDaemonAggregator_CompressedSizeRestrictedLarge(t *testing.T) {
	messages := []*stream.Message{
		mkTestMessage(t, strings.Repeat("1", 5_000), map[string]interface{}{}),
	}
	for i := 1; i < 30_000; i++ {
		messages = append(messages, messages[0])
	}

	belowSizeLimit := getAggregatorValidateFunction(65536)

	aggregatorTestCase{
		aggregationSize:    100_000,
		aggregationMaxSize: 65536,
		compression:        stream.CompressionGZip,
		messages:           messages,
		flushes: []expectedFlush{
			{
				messages: messages[0:5733],
				validate: belowSizeLimit,
			},
			{
				messages: messages[5733:12285],
				validate: belowSizeLimit,
			},
			{
				messages: messages[12285:18837],
				validate: belowSizeLimit,
			},
			{
				messages: messages[18837:25389],
				validate: belowSizeLimit,
			},
			{
				messages: messages[25389:],
				validate: belowSizeLimit,
			},
		},
	}.run(t)
}

func TestProducerDaemonAggregator_CompressedSizeRestrictedUncompressible(t *testing.T) {
	r := rand.NewSource(0x1020304050607080)

	messages := make([]*stream.Message, 0, 100)
	for i := 1; i < 100; i++ {
		var body strings.Builder
		for i := 0; i < 10_000; i++ {
			body.WriteByte(byte('A' + r.Int63()%('Z'-'A')))
		}

		message := mkTestMessage(t, body.String(), map[string]interface{}{})
		messages = append(messages, message)
	}

	belowSizeLimit := getAggregatorValidateFunction(65536)

	aggregatorTestCase{
		aggregationSize:    100_000,
		aggregationMaxSize: 65536,
		compression:        stream.CompressionGZip,
		messages:           messages,
		flushes: []expectedFlush{
			{
				messages: messages[0:8],
				validate: belowSizeLimit,
			},
			{
				messages: messages[8:16],
				validate: belowSizeLimit,
			},
			{
				messages: messages[16:24],
				validate: belowSizeLimit,
			},
			{
				messages: messages[24:32],
				validate: belowSizeLimit,
			},
			{
				messages: messages[32:40],
				validate: belowSizeLimit,
			},
			{
				messages: messages[40:48],
				validate: belowSizeLimit,
			},
			{
				messages: messages[48:56],
				validate: belowSizeLimit,
			},
			{
				messages: messages[56:64],
				validate: belowSizeLimit,
			},
			{
				messages: messages[64:72],
				validate: belowSizeLimit,
			},
			{
				messages: messages[72:80],
				validate: belowSizeLimit,
			},
			{
				messages: messages[80:88],
				validate: belowSizeLimit,
			},
			{
				messages: messages[88:96],
				validate: belowSizeLimit,
			},
			{
				messages: messages[96:],
				validate: belowSizeLimit,
			},
		},
	}.run(t)
}

func getAggregatorValidateFunction(sizeLimit int) func(flush *stream.AggregateFlush) error {
	return func(flush *stream.AggregateFlush) error {
		msg := stream.BuildAggregateMessage(flush.Body, flush.Attributes)

		encodedMsg, err := json.Marshal(msg)
		if err != nil {
			return err
		}

		if len(encodedMsg) > sizeLimit {
			return fmt.Errorf("message is too large: %d > %d", len(encodedMsg), sizeLimit)
		}

		return nil
	}
}
