package stream

import (
	"encoding/json"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/mon"
	"strings"
	"sync"
)

type kinsumerInput struct {
	config   cfg.Config
	logger   mon.Logger
	factory  KinsumerFactory
	settings KinsumerSettings
	client   Kinsumer
	channel  chan *Message
	wg       sync.WaitGroup
}

type KinsumerSettings struct {
	StreamName      string
	ApplicationName string
}

func NewKinsumerInput(config cfg.Config, logger mon.Logger, factory KinsumerFactory, settings KinsumerSettings) Input {
	client := factory(config, logger, settings)

	return &kinsumerInput{
		config:   config,
		logger:   logger,
		settings: settings,
		client:   client,
		channel:  make(chan *Message),
		factory:  factory,
	}
}

func (i *kinsumerInput) Data() chan *Message {
	return i.channel
}

func (i *kinsumerInput) Run() error {
	defer i.wg.Done()

	i.wg.Add(1)
	err := i.client.Run()

	if err != nil {
		i.logger.Fatal(err, "kinsumer.Kinsumer.Run() returned error")
	}

	for {
		rawMessage, err := i.client.Next()

		switch {
		case rawMessage == nil && err == nil: // kinsumer has been stopped
			close(i.channel)
			return nil

		case err != nil: // error occurred
			switch {
			case strings.Contains(err.Error(), "ExpiredIteratorException"):
				i.logger.WithFields(mon.Fields{
					"error": err,
				}).Warn("ExpiredIteratorException while consuming events")

				i.restartClient()

			case strings.Contains(err.Error(), "ConditionalCheckFailedException"):
				i.client.Stop()
				i.logger.WithFields(mon.Fields{
					"error": err,
				}).Warn("ConditionalCheckFailedException while consuming events")

			default:
				i.logger.Fatal(err, "Unexpected error while consuming events")
			}

		case rawMessage != nil: // rawMessage received
			msg := Message{}
			err := json.Unmarshal(rawMessage, &msg)

			if err != nil {
				i.logger.Error(err, "could not unmarshal message")
				continue
			}

			i.channel <- &msg
		}
	}
}

func (i *kinsumerInput) Stop() {
	i.client.Stop()
	i.wg.Wait()
}

func (i *kinsumerInput) restartClient() {
	var err error

	i.client.Stop()
	i.client = i.factory(i.config, i.logger, i.settings)

	if err = i.client.Run(); err != nil {
		i.logger.Fatal(err, "Unable to restart kinesis stream i")
	}

	i.logger.Info("Restarted kinesis stream i")
}
