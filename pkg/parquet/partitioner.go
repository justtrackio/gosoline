package parquet

import (
	"math"
	"sort"
	"sync"
	"time"

	"github.com/justtrackio/gosoline/pkg/clock"
)

type Partition struct {
	Timestamp     time.Time
	Elements      []Partitionable
	StartedAt     time.Time
	LastUpdatedAt time.Time
}

//go:generate go run github.com/vektra/mockery/v2 --name Partitioner
type Partitioner interface {
	Flush()
	Ingest(data Partitionable)
	Out() <-chan *Partition
	Size() int
	Start()
	Stop()
	Trim(size int)
}

type memoryPartitioner struct {
	lck           sync.Mutex
	clock         clock.Clock
	outputChannel chan *Partition
	ticker        *time.Ticker
	stop          chan struct{}

	partitions     map[float64]*Partition
	partitionCount int

	partitionInterval time.Duration
	bufferInterval    time.Duration
	maxPartitionSize  int
}

type PartitionerSettings struct {
	// at what granularity do we divide the data into partitions? Needs to be at least 1 second.
	PartitionInterval time.Duration `cfg:"partition_interval" default:"900s" validate:"min=1000000000"`
	// how long do we buffer elements before we write them out even when the partition
	// is not yet full. Needs to be at least 1 second.
	BufferInterval time.Duration `cfg:"buffer_interval" default:"900s" validate:"min=1000000000"`
	// how many elements can a partition have before we have to flush it (to avoid excessive memory usage)
	MaxPartitionSize int `cfg:"max_partition_size" default:"50000" validate:"min=1"`
}

func NewPartitioner(settings *PartitionerSettings) Partitioner {
	return NewPartitionerWithInterfaces(clock.Provider, settings)
}

func NewPartitionerWithInterfaces(clock clock.Clock, settings *PartitionerSettings) Partitioner {
	return &memoryPartitioner{
		clock:             clock,
		ticker:            time.NewTicker(settings.BufferInterval),
		partitionInterval: settings.PartitionInterval,
		bufferInterval:    settings.BufferInterval,
		maxPartitionSize:  settings.MaxPartitionSize,
		outputChannel:     make(chan *Partition),
		partitions:        map[float64]*Partition{},
		partitionCount:    0,
		stop:              make(chan struct{}),
	}
}

func (p *memoryPartitioner) Flush() {
	p.flush(true)
}

func (p *memoryPartitioner) Ingest(data Partitionable) {
	p.lck.Lock()
	defer p.lck.Unlock()

	timezone := data.GetPartitionTimestamp().Location()
	timestamp := data.GetPartitionTimestamp().Unix()

	partition := math.Floor(float64(timestamp) / p.partitionInterval.Seconds())
	partitionTimestamp := int64(partition * p.partitionInterval.Seconds())

	if _, ok := p.partitions[partition]; !ok {
		p.partitions[partition] = &Partition{
			Timestamp:     time.Unix(partitionTimestamp, 0).In(timezone),
			Elements:      make([]Partitionable, 0, 64),
			StartedAt:     p.clock.Now(),
			LastUpdatedAt: p.clock.Now(),
		}

		p.partitionCount++
	}

	p.partitions[partition].Elements = append(p.partitions[partition].Elements, data)
	p.partitions[partition].LastUpdatedAt = p.clock.Now()

	if len(p.partitions[partition].Elements) >= p.maxPartitionSize {
		p.outputChannel <- p.partitions[partition]

		delete(p.partitions, partition)
		p.partitionCount--
	}
}

func (p *memoryPartitioner) Out() <-chan *Partition {
	return p.outputChannel
}

func (p *memoryPartitioner) Size() int {
	return p.partitionCount
}

func (p *memoryPartitioner) Start() {
	go func() {
		for {
			select {
			case <-p.ticker.C:
				p.flush(false)
			case <-p.stop:
				return
			}
		}
	}()
}

func (p *memoryPartitioner) Stop() {
	p.ticker.Stop()
	close(p.stop)

	p.flush(true)
	close(p.outputChannel)
}

func (p *memoryPartitioner) Trim(size int) {
	p.lck.Lock()
	defer p.lck.Unlock()

	if p.partitionCount < size {
		return
	}

	lastUpdatedAts := make([]int64, 0, p.partitionCount)
	updatedAtToPartitionKey := make(map[int64]float64)

	for k, v := range p.partitions {
		nano := v.LastUpdatedAt.UnixNano()

		lastUpdatedAts = append(lastUpdatedAts, nano)
		updatedAtToPartitionKey[nano] = k
	}

	sort.Slice(lastUpdatedAts, func(i, j int) bool {
		return lastUpdatedAts[i] < lastUpdatedAts[j]
	})

	for i := 0; i < size; i++ {
		lastUpdatedAt := lastUpdatedAts[i]
		partitionKey := updatedAtToPartitionKey[lastUpdatedAt]

		p.outputChannel <- p.partitions[partitionKey]

		delete(p.partitions, partitionKey)
		p.partitionCount--
	}
}

func (p *memoryPartitioner) flush(force bool) {
	p.lck.Lock()
	defer p.lck.Unlock()

	now := p.clock.Now()

	for key, part := range p.partitions {
		age := now.Sub(part.StartedAt)

		if age < p.bufferInterval && !force {
			continue
		}

		p.outputChannel <- part

		delete(p.partitions, key)
		p.partitionCount--
	}
}
