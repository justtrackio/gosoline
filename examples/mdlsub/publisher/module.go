package main

import (
	"context"
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/clock"
	"github.com/applike/gosoline/pkg/kernel"
	"github.com/applike/gosoline/pkg/log"
	"github.com/applike/gosoline/pkg/mdlsub"
	"github.com/applike/gosoline/pkg/uuid"
	"math/rand"
	"time"
)

type RandomNumber struct {
	Id     string `json:"id"`
	Number int    `json:"number"`
}

type publisherModule struct {
	logger    log.Logger
	uuidGen   uuid.Uuid
	publisher mdlsub.Publisher
}

func newPublisherModule(_ context.Context, config cfg.Config, logger log.Logger) (kernel.Module, error) {
	var err error
	var publisher mdlsub.Publisher

	if publisher, err = mdlsub.NewPublisher(config, logger, "random-number"); err != nil {
		return nil, fmt.Errorf("can not create publisher random-number: %w", err)
	}

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

			p.logger.Info("published number %d", number.Number)
		}
	}
}
