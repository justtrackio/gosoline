package parquet_test

import (
	"github.com/applike/gosoline/pkg/parquet"
	"github.com/jonboulle/clockwork"
	"github.com/stretchr/testify/assert"
	"sync"
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
			TickerDuration: time.Duration(1) * time.Second,
		}
		parquet.NewPartitioner(settings)
	})
}

func TestMemoryPartitioner_Ingest(t *testing.T) {
	var lck sync.Mutex
	clock := clockwork.NewFakeClock()
	outputChan := make(chan *parquet.Partition)
	stop := make(chan struct{})

	partitioner := parquet.NewPartitionerWithInterfaces(lck, clock, 1*time.Hour, outputChan, stop)

	// now creating a new partition
	testData1 := &testDataType{
		CreatedAt: time.Unix(1499997599, 0),
	}

	// reusing the just created partition
	testData2 := &testDataType{
		CreatedAt: time.Unix(1499997600, 0),
	}

	// now creating a new partition
	testData3 := &testDataType{
		CreatedAt: time.Unix(1500000000, 0),
	}

	partitioner.Ingest(testData1)
	partitioner.Ingest(testData2)
	partitioner.Ingest(testData3)

	keys := partitioner.PartitionKeys()

	assert.Len(t, keys, 2)
	assert.Equal(t, float64(416665), keys[0])
	assert.Equal(t, float64(416666), keys[1])
}

func TestMemoryPartitioner_StartStop(t *testing.T) {
	settings := &parquet.PartitionerSettings{
		TickerDuration: time.Duration(1) * time.Second,
	}
	partitioner := parquet.NewPartitioner(settings)

	partitioner.Start()
	partitioner.Stop()
}

func TestMemoryPartitioner_StartDataFlushStop(t *testing.T) {
	var lck sync.Mutex
	clock := clockwork.NewFakeClock()
	outputChan := make(chan *parquet.Partition)
	stop := make(chan struct{})

	testData := &testDataType{
		CreatedAt: time.Unix(1499997599, 0),
	}

	partitioner := parquet.NewPartitionerWithInterfaces(lck, clock, 1*time.Hour, outputChan, stop)

	var partition *parquet.Partition

	go func() {
		partition = <-partitioner.Out()
	}()

	partitioner.Start()
	partitioner.Ingest(testData)
	partitioner.Stop()

	assert.NotNil(t, partition)
	assert.Len(t, partition.Elements, 1)
	assert.Equal(t, partition.Elements[0], testData)
}
