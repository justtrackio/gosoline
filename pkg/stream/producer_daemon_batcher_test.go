package stream_test

import (
	"encoding/json"
	"github.com/applike/gosoline/pkg/stream"
	"github.com/stretchr/testify/assert"
	"testing"
)

type batcherTestCase struct {
	batchMaxSize  int
	batchMaxBytes int

	messages []*stream.Message
	batches  [][]stream.WritableMessage
}

func (tc batcherTestCase) run(t *testing.T) {
	batcher := stream.NewProducerDaemonBatcher(stream.ProducerDaemonSettings{
		BatchSize:    tc.batchMaxSize,
		BatchMaxSize: tc.batchMaxBytes,
	})

	batches := make([][]stream.WritableMessage, 0)

	for _, msg := range tc.messages {
		batch, err := batcher.Append(msg)
		assert.NoError(t, err)

		if len(batch) > 0 {
			batches = append(batches, batch)
		}
	}

	batch := batcher.Flush()
	if len(batch) > 0 {
		batches = append(batches, batch)
	}

	assert.Len(t, batches, len(tc.batches))
	for i, batch := range batches {
		expectedBatch := tc.batches[i]
		assert.Len(t, batch, len(expectedBatch))
		batchLen := 0

		for j, msg := range batch {
			msgJson, err := json.Marshal(msg)
			assert.NoError(t, err)

			expectedMsgJson, err := json.Marshal(expectedBatch[j])
			assert.NoError(t, err)

			assert.JSONEq(t, string(expectedMsgJson), string(msgJson))
			batchLen += len(msgJson)
		}

		if tc.batchMaxBytes > 0 {
			assert.LessOrEqual(t, batchLen, tc.batchMaxBytes)
		}
	}
}

func TestProducerDaemonBatcher_Single(t *testing.T) {
	messages := []*stream.Message{
		mkTestMessage(t, "1", map[string]interface{}{}),
		mkTestMessage(t, "2", map[string]interface{}{}),
		mkTestMessage(t, "3", map[string]interface{}{}),
		mkTestMessage(t, "4", map[string]interface{}{}),
	}

	batcherTestCase{
		batchMaxSize:  1,
		batchMaxBytes: 0,
		messages:      messages,
		batches: [][]stream.WritableMessage{
			{
				messages[0],
			},
			{
				messages[1],
			},
			{
				messages[2],
			},
			{
				messages[3],
			},
		},
	}.run(t)
}

func TestProducerDaemonBatcher_SmallBatches(t *testing.T) {
	messages := []*stream.Message{
		mkTestMessage(t, "1", map[string]interface{}{}),
		mkTestMessage(t, "2", map[string]interface{}{}),
		mkTestMessage(t, "3", map[string]interface{}{}),
		mkTestMessage(t, "4", map[string]interface{}{}),
	}

	batcherTestCase{
		batchMaxSize:  3,
		batchMaxBytes: 0,
		messages:      messages,
		batches: [][]stream.WritableMessage{
			{
				messages[0],
				messages[1],
				messages[2],
			},
			{
				messages[3],
			},
		},
	}.run(t)
}

func TestProducerDaemonBatcher_SizeLimited(t *testing.T) {
	messages := []*stream.Message{
		mkTestMessage(t, "1", map[string]interface{}{}),
		mkTestMessage(t, "2", map[string]interface{}{}),
		mkTestMessage(t, "3", map[string]interface{}{}),
		mkTestMessage(t, "4", map[string]interface{}{}),
		mkTestMessage(t, "55", map[string]interface{}{}),
		mkTestMessage(t, "6", map[string]interface{}{}),
		mkTestMessage(t, "7", map[string]interface{}{}),
		mkTestMessage(t, "8", map[string]interface{}{}),
		mkTestMessage(t, "99", map[string]interface{}{}),
	}

	batcherTestCase{
		batchMaxSize:  10,
		batchMaxBytes: 122,
		messages:      messages,
		batches: [][]stream.WritableMessage{
			{
				messages[0],
				messages[1],
			},
			{
				messages[2],
				messages[3],
			},
			{
				messages[4],
			},
			{
				messages[5],
				messages[6],
			},
			{
				messages[7],
			},
			{
				messages[8],
			},
		},
	}.run(t)
}
