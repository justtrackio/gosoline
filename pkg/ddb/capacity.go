package ddb

import (
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

type Capacity struct {
	Total float64
	Read  float64
	Write float64
}

func (c *Capacity) add(cc *types.Capacity) {
	if cc == nil {
		return
	}

	if cc.CapacityUnits != nil {
		c.Total += *cc.CapacityUnits
	}

	if cc.CapacityUnits != nil {
		c.Read += *cc.ReadCapacityUnits
	}

	if cc.CapacityUnits != nil {
		c.Write += *cc.WriteCapacityUnits
	}
}

type ConsumedCapacity struct {
	Total float64
	Read  float64
	Write float64
	Table *Capacity
	LSI   map[string]*Capacity
	GSI   map[string]*Capacity
}

func newConsumedCapacity() *ConsumedCapacity {
	return &ConsumedCapacity{
		Table: &Capacity{},
		GSI:   make(map[string]*Capacity),
		LSI:   make(map[string]*Capacity),
	}
}

func (c *ConsumedCapacity) add(cc *types.ConsumedCapacity) {
	if cc == nil {
		return
	}

	if cc.CapacityUnits != nil {
		c.Total += *cc.CapacityUnits
	}

	if cc.ReadCapacityUnits != nil {
		c.Read += *cc.ReadCapacityUnits
	}

	if cc.WriteCapacityUnits != nil {
		c.Write += *cc.WriteCapacityUnits
	}

	if cc.Table != nil {
		c.Table.add(cc.Table)
	}

	if cc.LocalSecondaryIndexes != nil {
		for name, lsi := range cc.LocalSecondaryIndexes {
			if _, ok := c.LSI[name]; !ok {
				c.LSI[name] = &Capacity{}
			}

			c.LSI[name].add(&lsi)
		}
	}

	if cc.GlobalSecondaryIndexes != nil {
		for name, gsi := range cc.GlobalSecondaryIndexes {
			if _, ok := c.GSI[name]; !ok {
				c.GSI[name] = &Capacity{}
			}

			c.GSI[name].add(&gsi)
		}
	}
}

func (c *ConsumedCapacity) addSlice(capacities []types.ConsumedCapacity) {
	if capacities == nil {
		return
	}

	for _, cc := range capacities {
		c.add(&cc)
	}
}
