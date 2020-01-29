package parquet

import (
	"github.com/jonboulle/clockwork"
	"math"
	"sort"
	"sync"
	"time"
)

type Partition struct {
	Timestamp     time.Time
	Elements      []Partitionable
	StartedAt     time.Time
	LastUpdatedAt time.Time
}

//go:generate mockery -name Partitioner
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
	clock         clockwork.Clock
	outputChannel chan *Partition
	ticker        *time.Ticker
	stop          chan struct{}

	partitions     map[float64]*Partition
	partitionCount int
	interval       time.Duration
}

type PartitionerSettings struct {
	Interval time.Duration `cfg:"interval"`
}

func NewPartitioner(settings *PartitionerSettings) Partitioner {
	clock := clockwork.NewRealClock()

	return NewPartitionerWithInterfaces(clock, settings)
}

func NewPartitionerWithInterfaces(clock clockwork.Clock, settings *PartitionerSettings) Partitioner {
	return &memoryPartitioner{
		clock:          clock,
		ticker:         time.NewTicker(settings.Interval),
		interval:       settings.Interval,
		outputChannel:  make(chan *Partition),
		partitions:     map[float64]*Partition{},
		partitionCount: 0,
		stop:           make(chan struct{}),
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

	partition := math.Floor(float64(timestamp) / p.interval.Seconds())
	partitionTimestamp := int64(partition * p.interval.Seconds())

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
	now := p.clock.Now()

	for key, part := range p.partitions {
		age := now.Sub(part.StartedAt)

		if age < p.interval && !force {
			continue
		}

		p.outputChannel <- part

		delete(p.partitions, key)
		p.partitionCount--
	}
}
