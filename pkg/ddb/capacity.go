package ddb

import (
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/justtrackio/gosoline/pkg/mdl"
)

type capacityKind int

const (
	kindRead capacityKind = iota
	kindWrite
	kindMixed
)

type Capacity struct {
	kind  capacityKind
	total *float64
	read  *float64
	write *float64
}

func newCapacity(kind capacityKind) *Capacity {
	return &Capacity{
		kind: kind,
	}
}

func (c *Capacity) Total() float64 {
	if c.total != nil {
		return *c.total
	}

	return mdl.EmptyIfNil(c.read) + mdl.EmptyIfNil(c.write)
}

func (c *Capacity) Read() float64 {
	if c.read == nil {
		switch c.kind {
		case kindRead:
			return c.Total()
		case kindWrite:
			return 0.0
		}
	}

	return mdl.EmptyIfNil(c.read)
}

func (c *Capacity) Write() float64 {
	if c.write == nil {
		switch c.kind {
		case kindWrite:
			return c.Total()
		case kindRead:
			return 0.0
		}
	}

	return mdl.EmptyIfNil(c.write)
}

func (c *Capacity) add(cc *types.Capacity) {
	if cc == nil {
		return
	}

	c.addCapacity(mdl.EmptyIfNil(cc.CapacityUnits), mdl.EmptyIfNil(cc.ReadCapacityUnits), mdl.EmptyIfNil(cc.WriteCapacityUnits))
}

func (c *Capacity) addCapacity(total float64, read float64, write float64) {
	addSomeCapacity := func(field **float64, value float64) {
		if value != 0 {
			if *field == nil {
				*field = mdl.Box(value)
			} else {
				**field += value
			}
		}
	}

	addSomeCapacity(&c.total, total)
	addSomeCapacity(&c.read, read)
	addSomeCapacity(&c.write, write)
}

type ConsumedCapacity struct {
	Capacity
	Table Capacity
	LSI   map[string]*Capacity
	GSI   map[string]*Capacity
}

func newConsumedCapacity(kind capacityKind) *ConsumedCapacity {
	return &ConsumedCapacity{
		Capacity: *newCapacity(kind),
		Table:    *newCapacity(kind),
		GSI:      make(map[string]*Capacity),
		LSI:      make(map[string]*Capacity),
	}
}

func (c *ConsumedCapacity) add(cc *types.ConsumedCapacity) {
	if cc == nil {
		return
	}

	c.Capacity.add(&types.Capacity{
		CapacityUnits:      cc.CapacityUnits,
		ReadCapacityUnits:  cc.ReadCapacityUnits,
		WriteCapacityUnits: cc.WriteCapacityUnits,
	})

	c.Table.add(cc.Table)

	for name, lsi := range cc.LocalSecondaryIndexes {
		if _, ok := c.LSI[name]; !ok {
			c.LSI[name] = newCapacity(c.kind)
		}

		c.LSI[name].add(&lsi)
	}

	for name, gsi := range cc.GlobalSecondaryIndexes {
		if _, ok := c.GSI[name]; !ok {
			c.GSI[name] = newCapacity(c.kind)
		}

		c.GSI[name].add(&gsi)
	}
}

func (c *ConsumedCapacity) addSlice(capacities []types.ConsumedCapacity) {
	for _, cc := range capacities {
		c.add(&cc)
	}
}
