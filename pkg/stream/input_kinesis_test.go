package stream

import (
	"context"
	"testing"

	"github.com/justtrackio/gosoline/pkg/cloud/aws/kinesis"
	kinesisMocks "github.com/justtrackio/gosoline/pkg/cloud/aws/kinesis/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestKinesisInputRun(t *testing.T) {
	testCtx, cancel := context.WithCancel(context.Background())
	defer cancel()
	client := kinesisMocks.NewKinsumer(t)
	input := &kinesisInput{client: client}
	msgs := make([]*Message, 0, 2)
	acks := []bool{true, false}

	client.EXPECT().Run(testCtx, mock.Anything).Run(func(_ context.Context, process kinesis.RecordHandler) {
		assert.NoError(t, process(testCtx, []byte(`{"attributes":{"type":"message"},"body":"foo"}`)))
		assert.NoError(t, process(testCtx, []byte(`{"attributes":{"type":"message","version":0},"body":"foo"}`)))
		assert.Error(t, process(testCtx, []byte("not a message")))
	}).Return(nil).Once()

	err := input.Run(testCtx, func(ctx context.Context, msg *Message) bool {
		require.Same(t, testCtx, ctx)
		msgs = append(msgs, msg)

		return acks[len(msgs)-1]
	})

	assert.NoError(t, err)
	assert.Equal(t, []*Message{
		{
			Attributes: map[string]string{
				"type": "message",
			},
			Body: "foo",
		},
		{
			Attributes: map[string]string{
				"type":    "message",
				"version": "0",
			},
			Body: "foo",
		},
	}, msgs)
}

// TestKinesisInputRunReportsHandledRecordUnderCanceledContext guards against a record being consumed again although
// the consumer already dealt with it. Once process returned, the record was handled, so the kinsumer has to checkpoint
// it no matter whether it was acknowledged and no matter whether the context expired while it ran. Reporting the
// cancellation instead would stop the checkpoint before that record while the consumer has already put it into the
// retry queue, which delivers it twice.
func TestKinesisInputRunReportsHandledRecordUnderCanceledContext(t *testing.T) {
	testCtx, cancel := context.WithCancel(context.Background())
	defer cancel()
	client := kinesisMocks.NewKinsumer(t)
	input := &kinesisInput{client: client}

	client.EXPECT().Run(testCtx, mock.Anything).Run(func(_ context.Context, process kinesis.RecordHandler) {
		assert.NoError(t, process(testCtx, []byte(`{"attributes":{"type":"message"},"body":"foo"}`)))
	}).Return(nil).Once()

	processed := false
	err := input.Run(testCtx, func(ctx context.Context, _ *Message) bool {
		processed = true
		// The processing deadline expires while the callback runs and the callback declines to acknowledge.
		cancel()

		return false
	})

	assert.NoError(t, err)
	assert.True(t, processed)
}
