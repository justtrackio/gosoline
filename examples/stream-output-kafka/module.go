package main

import (
	"context"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/mon"
	"github.com/applike/gosoline/pkg/stream"
	"strconv"
	"sync"
)

type KafkaProducer struct {
	output stream.Output
}

func (k *KafkaProducer) Boot(config cfg.Config, logger mon.Logger) error {
	k.output = stream.NewKafkaOutput(logger, &stream.KafkaOutputSettings{
		Topic: "gosoline-example",
	})

	return nil
}

func (k *KafkaProducer) Run(ctx context.Context) error {
	size := 1000
	conc := 1000

	wg := sync.WaitGroup{}
	wg.Add(conc)

	for j := 0; j < conc; j++ {
		go func() {
			for i := 0; i < size; i++ {
				err := k.output.WriteOne(ctx, &stream.Message{
					Body: strconv.Itoa(i),
				})

				if err != nil {
					panic(err)
				}
			}

			wg.Done()
		}()
	}

	wg.Wait()

	return nil
}
