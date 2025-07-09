package retry_handler_sqs

import (
	"context"
	"fmt"
	"sync"

	"github.com/justtrackio/gosoline/pkg/funk"
	"github.com/justtrackio/gosoline/pkg/test/suite"
)

type Callback struct {
	aut                suite.AppUnderTest
	receivedModels     []DataModel
	receivedAttributes []map[string]string
	retryCount         int
	stopAt             int
	lck                sync.Mutex
}

func NewCallback() *Callback {
	return &Callback{}
}

func (c *Callback) Consume(_ context.Context, model DataModel, attributes map[string]string) (bool, error) {
	c.lck.Lock()
	defer c.lck.Unlock()

	// clone the attributes and remove the sqs specific ones to make for simpler checks later
	// we need to make a clone because deleting the message idea is a Bad Idea™ as it is needed to acknowledge the message
	attributes = funk.MergeMaps(attributes)
	delete(attributes, "sqsMessageId")
	delete(attributes, "sqsReceiptHandle")
	delete(attributes, "sqsApproximateReceiveCount")

	c.receivedModels = append(c.receivedModels, model)
	c.receivedAttributes = append(c.receivedAttributes, attributes)

	if len(c.receivedModels) <= c.retryCount {
		return false, fmt.Errorf("something went wrong on consume no %d", len(c.receivedModels))
	}

	if len(c.receivedModels) == c.stopAt {
		c.aut.Stop()
	}

	return true, nil
}
