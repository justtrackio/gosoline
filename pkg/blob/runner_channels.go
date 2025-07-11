package blob

import (
	"fmt"
	"sync"

	"github.com/justtrackio/gosoline/pkg/cfg"
)

var brc = struct {
	sync.Mutex
	instance *BatchRunnerChannels
}{}

func ProvideBatchRunnerChannels(config cfg.Config) (*BatchRunnerChannels, error) {
	brc.Lock()
	defer brc.Unlock()

	if brc.instance != nil {
		return brc.instance, nil
	}

	instance, err := NewBatchRunnerChannels(config)
	if err != nil {
		return nil, err
	}
	brc.instance = instance

	return brc.instance, nil
}

type BatchRunnerChannels struct {
	read   chan *Object
	write  chan *Object
	copy   chan *CopyObject
	delete chan *Object
}

func NewBatchRunnerChannels(config cfg.Config) (*BatchRunnerChannels, error) {
	settings := &BatchRunnerSettings{}
	if err := config.UnmarshalKey("blob", settings); err != nil {
		return nil, fmt.Errorf("failed to unmarshal batch runner settings: %w", err)
	}

	return &BatchRunnerChannels{
		read:   make(chan *Object, settings.ReaderRunnerCount),
		write:  make(chan *Object, settings.WriterRunnerCount),
		copy:   make(chan *CopyObject, settings.CopyRunnerCount),
		delete: make(chan *Object, settings.DeleteRunnerCount),
	}, nil
}
