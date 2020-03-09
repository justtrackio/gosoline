package stream

import (
	"context"
)

type multiProducer []Producer

func NewMultiProducer(producers ...Producer) Producer {
	return multiProducer(producers)
}

func (mp multiProducer) WriteOne(ctx context.Context, model interface{}, attributeSets ...map[string]interface{}) error {
	for _, p := range mp {
		err := p.WriteOne(ctx, model, attributeSets...)
		if err != nil {
			return err
		}
	}

	return nil
}

func (mp multiProducer) Write(ctx context.Context, models interface{}, attributeSets ...map[string]interface{}) error {
	for _, p := range mp {
		err := p.Write(ctx, models, attributeSets...)
		if err != nil {
			return err
		}
	}

	return nil
}
