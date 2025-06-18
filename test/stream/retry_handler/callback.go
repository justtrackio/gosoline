package retry_handler

import (
	"context"
	"fmt"

	"github.com/justtrackio/gosoline/pkg/test/suite"
)

type DataModel struct {
	Id    string
	Title string
}

type Callback struct {
	aut                suite.AppUnderTest
	receivedModels     []DataModel
	receivedAttributes []map[string]string
}

func NewCallback() *Callback {
	return &Callback{}
}

func (c *Callback) GetModel(attributes map[string]string) interface{} {
	return &DataModel{}
}

func (c *Callback) Consume(ctx context.Context, model DataModel, attributes map[string]string) (bool, error) {
	c.receivedModels = append(c.receivedModels, model)
	c.receivedAttributes = append(c.receivedAttributes, attributes)

	if len(c.receivedModels) < 3 {
		return false, fmt.Errorf("something went wrong on consume no %d", len(c.receivedModels))
	}

	c.aut.Stop()

	return true, nil
}
