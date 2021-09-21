package blob

import (
	"sync"

	"github.com/justtrackio/gosoline/pkg/cfg"
)

var brc = struct {
	sync.Mutex
	instance *BatchRunnerChannels
}{}

func ProvideBatchRunnerChannels(config cfg.Config) *BatchRunnerChannels {
	brc.Lock()
	defer brc.Unlock()

	if brc.instance != nil {
		return brc.instance
	}

	brc.instance = NewBatchRunnerChannels(config)

	return brc.instance
}

type BatchRunnerChannels struct {
	read   chan *Object
	write  chan *Object
	copy   chan *CopyObject
	delete chan *Object
}

func NewBatchRunnerChannels(config cfg.Config) *BatchRunnerChannels {
	settings := &BatchRunnerSettings{}
	config.UnmarshalKey("blob", settings)

	return &BatchRunnerChannels{
		read:   make(chan *Object, settings.ReaderRunnerCount),
		write:  make(chan *Object, settings.WriterRunnerCount),
		copy:   make(chan *CopyObject, settings.CopyRunnerCount),
		delete: make(chan *Object, settings.DeleteRunnerCount),
	}
}
