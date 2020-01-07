package parquet

import (
	"github.com/jonboulle/clockwork"
	"math"
	"sync"
	"time"
)

type Partition struct {
	StartedAt time.Time
	Timestamp time.Time
	Elements  []Partitionable
}

//go:generate mockery -name Partitioner
type Partitioner interface {
	Flush()
	Ingest(data Partitionable)
	Out() <-chan *Partition
	Start()
	Stop()
}

type memoryPartitioner struct {
	lck           sync.Mutex
	clock         clockwork.Clock
	outputChannel chan *Partition
	ticker        *time.Ticker
	stop          chan struct{}

	partitions map[float64]*Partition
	interval   time.Duration
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
		clock:         clock,
		ticker:        time.NewTicker(settings.Interval),
		interval:      settings.Interval,
		outputChannel: make(chan *Partition),
		partitions:    map[float64]*Partition{},
		stop:          make(chan struct{}),
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
			StartedAt: p.clock.Now(),
			Timestamp: time.Unix(partitionTimestamp, 0).In(timezone),
			Elements:  make([]Partitionable, 0, 64),
		}
	}

	p.partitions[partition].Elements = append(p.partitions[partition].Elements, data)
}

func (p *memoryPartitioner) Out() <-chan *Partition {
	return p.outputChannel
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

func (p *memoryPartitioner) flush(force bool) {
	p.lck.Lock()
	defer p.lck.Unlock()

	now := p.clock.Now()

	for key, part := range p.partitions {
		age := now.Sub(part.StartedAt)

		if age < p.interval && !force {
			continue
		}

		p.outputChannel <- part

		delete(p.partitions, key)
	}
}
