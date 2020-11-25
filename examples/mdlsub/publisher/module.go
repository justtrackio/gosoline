package main

import (
	"context"
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/clock"
	"github.com/applike/gosoline/pkg/kernel"
	"github.com/applike/gosoline/pkg/mdlsub"
	"github.com/applike/gosoline/pkg/mon"
	"github.com/applike/gosoline/pkg/uuid"
	"math/rand"
	"time"
)

type RandomNumber struct {
	Id     string `json:"id"`
	Number int    `json:"number"`
}

type publisherModule struct {
	logger    mon.Logger
	uuidGen   uuid.Uuid
	publisher mdlsub.Publisher
}

func newPublisherModule(_ context.Context, config cfg.Config, logger mon.Logger) (kernel.Module, error) {
	publisher := mdlsub.NewPublisher(config, logger, "random-number")

	module := &publisherModule{
		logger:    logger,
		uuidGen:   uuid.New(),
		publisher: publisher,
	}

	return module, nil
}

func (p publisherModule) Run(ctx context.Context) error {
	ticker := clock.NewRealTicker(time.Second)

	for {
		select {
		case <-ctx.Done():
			return nil

		case <-ticker.Tick():
			number := RandomNumber{
				Id:     p.uuidGen.NewV4(),
				Number: rand.Intn(100),
			}

			if err := p.publisher.Publish(ctx, mdlsub.TypeCreate, 0, number); err != nil {
				return fmt.Errorf("can not publish random number %d: %w", number.Number, err)
			}

			p.logger.Infof("published number %d", number.Number)
		}
	}
}
