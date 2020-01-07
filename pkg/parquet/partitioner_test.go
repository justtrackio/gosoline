package parquet_test

import (
	"github.com/applike/gosoline/pkg/parquet"
	"github.com/jonboulle/clockwork"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
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
			Interval: time.Duration(1) * time.Second,
		}
		parquet.NewPartitioner(settings)
	})
}

func TestMemoryPartitioner_Ingest(t *testing.T) {
	clock := clockwork.NewFakeClockAt(time.Unix(1578500000, 0))
	partitioner := parquet.NewPartitionerWithInterfaces(clock, &parquet.PartitionerSettings{
		Interval: 2 * time.Second,
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

	clock.Advance(5 * time.Second)

	go partitioner.Stop()

	partitions := make([]*parquet.Partition, 0)

	for part := range partitioner.Out() {
		partitions = append(partitions, part)
	}

	assert.Len(t, partitions, 2)
}

func TestMemoryPartitioner_StartStop(t *testing.T) {
	settings := &parquet.PartitionerSettings{
		Interval: time.Duration(1) * time.Second,
	}
	partitioner := parquet.NewPartitioner(settings)

	partitioner.Start()
	partitioner.Stop()
}
