package main

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/clock"
	"github.com/justtrackio/gosoline/pkg/kernel"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/mdlsub"
	"github.com/justtrackio/gosoline/pkg/uuid"
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

func newPublisherModule(ctx context.Context, config cfg.Config, logger log.Logger) (kernel.Module, error) {
	var err error
	var publisher mdlsub.Publisher

	if publisher, err = mdlsub.NewPublisher(ctx, config, logger, "random-number"); err != nil {
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
