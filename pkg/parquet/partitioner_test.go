package parquet_test

import (
	"testing"
	"time"

	"github.com/justtrackio/gosoline/pkg/clock"
	"github.com/justtrackio/gosoline/pkg/parquet"
	"github.com/stretchr/testify/assert"
)

type testDataType struct {
	CreatedAt time.Time
}

func (t *testDataType) GetPartitionTimestamp() time.Time {
	return t.CreatedAt
}

func TestNewPartitioner(t *testing.T) {
	assert.NotPanics(t, func() {
		settings := &parquet.PartitionerSettings{
			PartitionInterval: time.Second,
			BufferInterval:    time.Second,
			MaxPartitionSize:  10,
		}
		parquet.NewPartitioner(settings)
	})
}

func TestMemoryPartitioner_Ingest(t *testing.T) {
	testClock := clock.NewFakeClockAt(time.Unix(1578500000, 0))
	partitioner := parquet.NewPartitionerWithInterfaces(testClock, &parquet.PartitionerSettings{
		PartitionInterval: 2 * time.Second,
		BufferInterval:    2 * time.Second,
		MaxPartitionSize:  10,
	})

	// now creating a new partition
	testData1 := &testDataType{
		CreatedAt: time.Unix(1578500000, 0),
	}

	// creating a second partition
	testData2 := &testDataType{
		CreatedAt: time.Unix(1578500001, 0),
	}

	// reusing the second partition
	testData3 := &testDataType{
		CreatedAt: time.Unix(1578500003, 0),
	}

	partitioner.Ingest(testData1)
	partitioner.Ingest(testData2)
	partitioner.Ingest(testData3)

	testClock.Advance(5 * time.Second)

	go partitioner.Stop()

	partitions := make([]*parquet.Partition, 0)

	for part := range partitioner.Out() {
		partitions = append(partitions, part)
	}

	assert.Len(t, partitions, 2)
}

func TestMemoryPartitioner_StartStop(t *testing.T) {
	settings := &parquet.PartitionerSettings{
		PartitionInterval: time.Second,
		BufferInterval:    time.Second,
		MaxPartitionSize:  10,
	}
	partitioner := parquet.NewPartitioner(settings)

	partitioner.Start()
	partitioner.Stop()
}

type testEvent struct {
	id        int
	createdAt time.Time
}

func (t testEvent) GetPartitionTimestamp() time.Time {
	return t.createdAt
}

func TestMemoryPartitioner_ReceivesAll(t *testing.T) {
	settings := &parquet.PartitionerSettings{
		PartitionInterval: time.Millisecond * 10,
		BufferInterval:    time.Millisecond * 10,
		MaxPartitionSize:  10_000,
	}
	partitioner := parquet.NewPartitioner(settings)
	partitioner.Start()

	c := make(chan int)
	seenEvents := make(map[int]struct{})

	go func() {
		elementCount := 0
		partitionCount := 0

		for batch := range partitioner.Out() {
			elementCount += len(batch.Elements)
			partitionCount++

			for _, elem := range batch.Elements {
				seenEvents[elem.(*testEvent).id] = struct{}{}
			}
		}

		c <- elementCount
		c <- partitionCount
	}()

	totalEvents := 100_000

	for i := 0; i < totalEvents; i++ {
		partitioner.Ingest(&testEvent{
			id:        i,
			createdAt: time.Now(),
		})
	}

	partitioner.Stop()

	partitionedElements := <-c
	partitionCount := <-c
	assert.Equal(t, totalEvents, partitionedElements, "expected every element to get returned")
	assert.Less(t, partitionCount, 100, "expected less than 100 partitions")
	assert.Equal(t, totalEvents, len(seenEvents), "expected each element to be returned exactly once")
}
