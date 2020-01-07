package parquet

import (
	"github.com/jonboulle/clockwork"
	"github.com/thoas/go-funk"
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
	Ingest(data Partitionable)
	PartitionKeys() []float64
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

// TODO: we might later also want to add a "maxPartitionCount" and a "maxPartitionSize" settings to force write the file and not get into a memory problem
type PartitionerSettings struct {
	TickerDuration time.Duration `cfg:"tickerDuration"`
}

func NewPartitioner(settings *PartitionerSettings) Partitioner {
	var lck sync.Mutex
	clock := clockwork.NewRealClock()
	outputChan := make(chan *Partition) // TODO: should we make this buffered?
	stop := make(chan struct{})

	return NewPartitionerWithInterfaces(lck, clock, settings.TickerDuration, outputChan, stop)
}

func NewPartitionerWithInterfaces(lck sync.Mutex, clock clockwork.Clock, duration time.Duration, outputChan chan *Partition, stop chan struct{}) Partitioner {
	return &memoryPartitioner{
		lck:           lck,
		clock:         clock,
		ticker:        time.NewTicker(duration),
		interval:      duration,
		outputChannel: outputChan,
		partitions:    map[float64]*Partition{},
		stop:          stop,
	}
}

func (p *memoryPartitioner) Ingest(data Partitionable) {
	p.lck.Lock()
	defer p.lck.Unlock()

	timestamp := data.GetPartitionTimestamp().Unix()

	partition := math.Floor(float64(timestamp) / p.interval.Seconds())
	partitionTimestamp := int64(partition * p.interval.Seconds())

	if _, ok := p.partitions[partition]; !ok {
		p.partitions[partition] = &Partition{
			StartedAt: p.clock.Now(),
			Timestamp: time.Unix(partitionTimestamp, 0),
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

func (p *memoryPartitioner) PartitionKeys() []float64 {
	p.lck.Lock()
	defer p.lck.Unlock()

	return funk.Keys(p.partitions).([]float64)
}
