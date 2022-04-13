package consumer_test

import (
	"context"
	"testing"
	"time"

	"github.com/justtrackio/gosoline/pkg/kafka/consumer"
	"github.com/justtrackio/gosoline/pkg/test/assert"
	"github.com/segmentio/kafka-go"
)

func TestBatcher(t *testing.T) {
	type args struct {
		batchSize    int
		batchTimeout time.Duration
		input        func() chan kafka.Message
		ctx          func() (context.Context, func())
	}

	tests := []struct {
		name string
		args args
		want []kafka.Message
	}{
		{
			name: "batch time out reached",
			args: args{
				batchSize:    10,
				batchTimeout: time.Second,
				ctx:          func() (context.Context, func()) { return context.Background(), func() {} },
				input: func() chan kafka.Message {
					c := make(chan kafka.Message, 5)
					c <- kafka.Message{Partition: 0, Offset: 0}
					c <- kafka.Message{Partition: 1, Offset: 1}
					c <- kafka.Message{Partition: 2, Offset: 2}
					c <- kafka.Message{Partition: 3, Offset: 3}
					c <- kafka.Message{Partition: 4, Offset: 4}

					return c
				},
			},
			want: []kafka.Message{
				{Partition: 0, Offset: 0},
				{Partition: 1, Offset: 1},
				{Partition: 2, Offset: 2},
				{Partition: 3, Offset: 3},
				{Partition: 4, Offset: 4},
			},
		},
		{
			name: "batch time out reached multiple times",
			args: args{
				batchSize:    10,
				batchTimeout: 100 * time.Millisecond,
				ctx:          func() (context.Context, func()) { return context.Background(), func() {} },
				input: func() chan kafka.Message {
					c := make(chan kafka.Message, 5)

					go func() {
						time.Sleep(600 * time.Millisecond)

						c <- kafka.Message{Partition: 0, Offset: 0}
						c <- kafka.Message{Partition: 1, Offset: 1}
						c <- kafka.Message{Partition: 2, Offset: 2}
						c <- kafka.Message{Partition: 3, Offset: 3}

						time.Sleep(time.Second)

						c <- kafka.Message{Partition: 4, Offset: 4}
					}()

					return c
				},
			},
			want: []kafka.Message{
				{Partition: 0, Offset: 0},
				{Partition: 1, Offset: 1},
				{Partition: 2, Offset: 2},
				{Partition: 3, Offset: 3},
			},
		},
		{
			name: "batch size reached",
			args: args{
				batchSize:    5,
				batchTimeout: time.Hour,
				ctx:          func() (context.Context, func()) { return context.Background(), func() {} },
				input: func() chan kafka.Message {
					c := make(chan kafka.Message, 5)
					c <- kafka.Message{Partition: 0, Offset: 0}
					c <- kafka.Message{Partition: 1, Offset: 1}
					c <- kafka.Message{Partition: 2, Offset: 2}
					c <- kafka.Message{Partition: 3, Offset: 3}
					c <- kafka.Message{Partition: 4, Offset: 4}

					return c
				},
			},
			want: []kafka.Message{
				{Partition: 0, Offset: 0},
				{Partition: 1, Offset: 1},
				{Partition: 2, Offset: 2},
				{Partition: 3, Offset: 3},
				{Partition: 4, Offset: 4},
			},
		},
		{
			name: "batch size reached (batch timeout too small)",
			args: args{
				batchSize:    10,
				batchTimeout: time.Microsecond,
				ctx:          func() (context.Context, func()) { return context.Background(), func() {} },
				input: func() chan kafka.Message {
					c := make(chan kafka.Message, 5)
					c <- kafka.Message{Partition: 0, Offset: 0}
					c <- kafka.Message{Partition: 1, Offset: 1}
					c <- kafka.Message{Partition: 2, Offset: 2}
					c <- kafka.Message{Partition: 3, Offset: 3}
					c <- kafka.Message{Partition: 4, Offset: 4}

					return c
				},
			},
			want: []kafka.Message{
				{Partition: 0, Offset: 0},
				{Partition: 1, Offset: 1},
				{Partition: 2, Offset: 2},
				{Partition: 3, Offset: 3},
				{Partition: 4, Offset: 4},
			},
		},
		{
			name: "context canceled (no messages)",
			args: args{
				batchSize:    10,
				batchTimeout: time.Hour,
				ctx: func() (context.Context, func()) {
					return context.WithTimeout(context.Background(), time.Second) //nolint
				},
				input: func() chan kafka.Message {
					c := make(chan kafka.Message, 5)
					return c
				},
			},
			want: []kafka.Message{},
		},
		{
			name: "context canceled (mid batch)",
			args: args{
				batchSize:    4,
				batchTimeout: time.Hour,
				ctx: func() (context.Context, func()) {
					return context.WithTimeout(context.Background(), time.Second) //nolint
				},
				input: func() chan kafka.Message {
					c := make(chan kafka.Message, 5)
					c <- kafka.Message{Partition: 0, Offset: 0}
					c <- kafka.Message{Partition: 1, Offset: 1}

					return c
				},
			},
			want: []kafka.Message{
				{Partition: 0, Offset: 0},
				{Partition: 1, Offset: 1},
			},
		},
		{
			name: "incoming > batch size",
			args: args{
				batchSize:    1,
				batchTimeout: time.Hour,
				ctx:          func() (context.Context, func()) { return context.Background(), func() {} },
				input: func() chan kafka.Message {
					c := make(chan kafka.Message, 5)
					c <- kafka.Message{Partition: 0, Offset: 0}
					c <- kafka.Message{Partition: 1, Offset: 1}
					c <- kafka.Message{Partition: 2, Offset: 2}
					c <- kafka.Message{Partition: 3, Offset: 3}
					c <- kafka.Message{Partition: 4, Offset: 4}

					return c
				},
			},
			want: []kafka.Message{
				{Partition: 0, Offset: 0},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := tt.args.ctx()
			defer cancel()

			batcher := consumer.NewBatcher(tt.args.input(), tt.args.batchSize, tt.args.batchTimeout)
			assert.Equal(t, tt.want, batcher.Get(ctx))
		})
	}

}
