package blob

import (
	"context"
	"fmt"

	"github.com/justtrackio/gosoline/pkg/appctx"
	"github.com/justtrackio/gosoline/pkg/cfg"
)

type blobRunnerChannelsKey string

func ProvideBatchRunnerChannels(ctx context.Context, config cfg.Config, name string) (*BatchRunnerChannels, error) {
	return appctx.Provide(ctx, blobRunnerChannelsKey(name), func() (*BatchRunnerChannels, error) {
		return NewBatchRunnerChannels(config, name)
	})
}

type BatchRunnerChannels struct {
	read   chan *Object
	write  chan *Object
	copy   chan *CopyObject
	delete chan *Object
}

func NewBatchRunnerChannels(config cfg.Config, name string) (*BatchRunnerChannels, error) {
	var settings BatchRunnerSettings
	var err error

	configKey := getConfigKey(name)

	if settings, err = cfg.GetSettings[BatchRunnerSettings](config, configKey); err != nil {
		return nil, fmt.Errorf("failed to unmarshal batch runner settings: %w", err)
	}

	return &BatchRunnerChannels{
		read:   make(chan *Object, settings.ReaderRunnerCount),
		write:  make(chan *Object, settings.WriterRunnerCount),
		copy:   make(chan *CopyObject, settings.CopyRunnerCount),
		delete: make(chan *Object, settings.DeleteRunnerCount),
	}, nil
}
