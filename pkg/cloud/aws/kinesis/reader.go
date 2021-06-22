package kinesis

import (
	"context"
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/log"
	"strings"
	"sync"
)

//go:generate mockery -name Reader
type Reader interface {
	Run(ctx context.Context) error
	Stop()
}

type KinsumerFactory func(config cfg.Config, logger log.Logger, settings KinsumerSettings) (Kinsumer, error)

type reader struct {
	config   cfg.Config
	logger   log.Logger
	factory  KinsumerFactory
	settings KinsumerSettings
	client   Kinsumer
	handler  MessageHandler
	doStop   sync.Once
	wg       sync.WaitGroup
}

func NewReader(config cfg.Config, logger log.Logger, factory KinsumerFactory, handler MessageHandler, settings KinsumerSettings) (Reader, error) {
	client, err := factory(config, logger, settings)

	if err != nil {
		return nil, fmt.Errorf("unable to create kinesis client: %w", err)
	}

	return &reader{
		config:   config,
		logger:   logger,
		settings: settings,
		client:   client,
		handler:  handler,
		factory:  factory,
	}, nil
}

func (r *reader) Run(ctx context.Context) error {
	defer r.handler.Done()

	r.wg.Add(1)
	defer r.wg.Done()

	logger := r.logger.WithContext(ctx)

	err := r.client.Run()

	if err != nil {
		return fmt.Errorf("kinsumer.Kinsumer.Run() returned error %w", err)
	}

	for {
		rawMessage, err := r.client.Next()

		if err != nil {
			errMsg := err.Error()

			switch {
			case strings.Contains(errMsg, "ExpiredIteratorException"):
				logger.WithFields(log.Fields{
					"error": errMsg,
				}).Warn("ExpiredIteratorException while consuming events")

				if err := r.restartClient(); err != nil {
					return err
				}

			case strings.Contains(errMsg, "ConditionalCheckFailedException"):
				r.stopClient()
				logger.WithFields(log.Fields{
					"error": errMsg,
				}).Warn("ConditionalCheckFailedException while consuming events")

			default:
				return fmt.Errorf("unexpected error while consuming events: %w", err)
			}

			continue
		}

		if rawMessage == nil {
			// kinsumer has been stopped
			return nil
		}

		// rawMessage received
		if err := r.handler.Handle(rawMessage); err != nil {
			logger.Error("could not handle message: %w", err)
		}
	}
}

func (r *reader) Stop() {
	r.stopClient()
	r.wg.Wait()
}

func (r *reader) stopClient() {
	r.doStop.Do(func() {
		r.client.Stop()
	})
}

func (r *reader) restartClient() error {
	r.stopClient()

	var err error
	r.client, err = r.factory(r.config, r.logger, r.settings)

	if err != nil {
		return fmt.Errorf("unable to create new kinesis client: %w", err)
	}

	// allow us to stop the new client again
	r.doStop = sync.Once{}

	if err := r.client.Run(); err != nil {
		return fmt.Errorf("unable to restart kinesis stream input: %w", err)
	}

	r.logger.Info("Restarted kinesis stream input")

	return nil
}
