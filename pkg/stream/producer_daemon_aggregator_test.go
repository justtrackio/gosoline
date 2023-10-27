package stream_test

import (
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"testing"

	"github.com/justtrackio/gosoline/pkg/encoding/base64"
	"github.com/justtrackio/gosoline/pkg/encoding/json"
	"github.com/justtrackio/gosoline/pkg/stream"
	"github.com/stretchr/testify/assert"
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
		flushList, err := agg.Write(context.Background(), msg)
		assert.NoError(t, err)

		flushes = append(flushes, flushList...)
	}

	newFlushes, err := agg.Flush()
	assert.NoError(t, err)
	for _, flush := range newFlushes {
		if flush.MessageCount > 0 {
			flushes = append(flushes, flush)
		}
	}

	assert.Len(t, flushes, len(tc.flushes))

	expectedFlushes := make([]stream.AggregateFlush, len(tc.flushes))
	for i, expectedFlush := range tc.flushes {
		expectedFlushes[i] = expectedFlush.encode(tc.compression)

		if tc.aggregationMaxSize > 0 && len(flushes) > i {
			assert.LessOrEqualf(t, len(flushes[i].Body), tc.aggregationMaxSize, fmt.Sprintf("message is too large: %d > %d", len(flushes[i].Body), tc.aggregationMaxSize))
		}
	}

	assert.Equal(t, expectedFlushes, flushes)
}

type expectedFlush struct {
	messages []*stream.Message
}

func (f expectedFlush) encode(compression stream.CompressionType) stream.AggregateFlush {
	body, err := json.Marshal(f.messages)
	if err != nil {
		panic(err)
	}

	attributes := map[string]string{
		stream.AttributeEncoding:       stream.EncodingJson.String(),
		stream.AttributeAggregateCount: strconv.Itoa(len(f.messages)),
	}

	if compression == stream.CompressionGZip {
		attributes[stream.AttributeCompression] = compression.String()

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

func mkTestMessage(t *testing.T, body interface{}, attributes map[string]string) *stream.Message {
	attributes[stream.AttributeEncoding] = stream.EncodingJson.String()

	bodyBytes, err := json.Marshal(body)
	assert.NoError(t, err)

	return &stream.Message{
		Attributes: attributes,
		Body:       string(bodyBytes),
	}
}

func TestProducerDaemonAggregator_CountRestricted(t *testing.T) {
	messages := []*stream.Message{
		mkTestMessage(t, "message 1", map[string]string{}),
		mkTestMessage(t, "message 2", map[string]string{}),
		mkTestMessage(t, "message 3", map[string]string{
			"attribute": "value",
		}),
		mkTestMessage(t, "message 4", map[string]string{}),
		mkTestMessage(t, "message 5", map[string]string{}),
		mkTestMessage(t, "message 6", map[string]string{
			"attribute": "another value",
		}),
		mkTestMessage(t, "message 7", map[string]string{}),
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
		mkTestMessage(t, strings.Repeat("1", 50), map[string]string{}),
		mkTestMessage(t, strings.Repeat("2", 50), map[string]string{}),
		mkTestMessage(t, strings.Repeat("3", 50), map[string]string{
			"attribute": fmt.Sprintf("l%sng value", strings.Repeat("o", 100)),
		}),
		mkTestMessage(t, strings.Repeat("4", 50), map[string]string{}),
		mkTestMessage(t, strings.Repeat("5", 50), map[string]string{}),
		mkTestMessage(t, strings.Repeat("6", 50), map[string]string{
			"attribute": "another value",
		}),
		mkTestMessage(t, strings.Repeat("7", 50), map[string]string{}),
	}

	aggregatorTestCase{
		aggregationSize:    5_000,
		aggregationMaxSize: 300,
		compression:        stream.CompressionNone,
		messages:           messages,
		flushes: []expectedFlush{
			{
				messages: messages[0:2],
			},
			{
				messages: messages[2:3],
			},
			{
				messages: messages[3:5],
			},
			{
				messages: messages[5:],
			},
		},
	}.run(t)
}

func TestProducerDaemonAggregator_CompressedSizeRestrictedSmall(t *testing.T) {
	messages := []*stream.Message{
		mkTestMessage(t, strings.Repeat("1", 5_000), map[string]string{}),
		mkTestMessage(t, strings.Repeat("2", 5_000), map[string]string{}),
		mkTestMessage(t, strings.Repeat("3", 5_000), map[string]string{
			"attribute": fmt.Sprintf("l%sng value", strings.Repeat("o", 100)),
		}),
		mkTestMessage(t, strings.Repeat("4", 5_000), map[string]string{}),
		mkTestMessage(t, strings.Repeat("5", 5_000), map[string]string{}),
		mkTestMessage(t, strings.Repeat("6", 5_000), map[string]string{
			"attribute": "another value",
		}),
		mkTestMessage(t, strings.Repeat("7", 5_000), map[string]string{}),
	}

	aggregatorTestCase{
		aggregationSize:    5_000,
		aggregationMaxSize: 5_000,
		compression:        stream.CompressionGZip,
		messages:           messages,
		flushes: []expectedFlush{
			{
				messages: messages[0:1],
			},
			{
				messages: messages[1:],
			},
		},
	}.run(t)
}

func TestProducerDaemonAggregator_CompressedSizeRestrictedLarge(t *testing.T) {
	messages := []*stream.Message{
		mkTestMessage(t, strings.Repeat("1", 3_000), map[string]string{}),
	}
	for i := 1; i < 3_000; i++ {
		messages = append(messages, messages[0])
	}

	aggregatorTestCase{
		aggregationSize:    10_000,
		aggregationMaxSize: 16000,
		compression:        stream.CompressionGZip,
		messages:           messages,
		flushes: []expectedFlush{
			{
				messages: messages[:1362],
			},
			{
				messages: messages[1362:2724],
			},
			{
				messages: messages[2724:],
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

		message := mkTestMessage(t, body.String(), map[string]string{})
		messages = append(messages, message)
	}

	aggregatorTestCase{
		aggregationSize:    100_000,
		aggregationMaxSize: 65536,
		compression:        stream.CompressionGZip,
		messages:           messages,
		flushes: []expectedFlush{
			{
				messages: messages[0:8],
			},
			{
				messages: messages[8:16],
			},
			{
				messages: messages[16:24],
			},
			{
				messages: messages[24:32],
			},
			{
				messages: messages[32:40],
			},
			{
				messages: messages[40:48],
			},
			{
				messages: messages[48:56],
			},
			{
				messages: messages[56:64],
			},
			{
				messages: messages[64:72],
			},
			{
				messages: messages[72:80],
			},
			{
				messages: messages[80:88],
			},
			{
				messages: messages[88:96],
			},
			{
				messages: messages[96:],
			},
		},
	}.run(t)
}
