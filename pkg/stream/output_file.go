package stream

import (
	"context"
	"fmt"
	"os"
	"sync"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
)

type FileOutputMode string

const (
	FileOutputModeAppend   FileOutputMode = "append"
	FileOutputModeSingle   FileOutputMode = "single"
	FileOutputModeTruncate FileOutputMode = "truncate"
)

type FileOutputSettings struct {
	Filename string         `cfg:"filename"`
	Mode     FileOutputMode `cfg:"mode"     default:"append"`
}

type fileOutput struct {
	logger   log.Logger
	settings *FileOutputSettings
	lck      sync.Mutex
	cnt      int
}

func NewFileOutput(_ cfg.Config, logger log.Logger, settings *FileOutputSettings) Output {
	return &fileOutput{
		logger:   logger,
		settings: settings,
	}
}

func (o *fileOutput) WriteOne(ctx context.Context, msg WritableMessage) error {
	return o.Write(ctx, []WritableMessage{msg})
}

func (o *fileOutput) Write(_ context.Context, batch []WritableMessage) error {
	o.lck.Lock()
	defer o.lck.Unlock()

	filename := o.settings.Filename
	flags := os.O_CREATE | os.O_WRONLY

	switch o.settings.Mode {
	case FileOutputModeSingle:
		filename = fmt.Sprintf("%s-%d", filename, o.cnt)
		flags |= os.O_TRUNC
		o.cnt++
	case FileOutputModeTruncate:
		flags |= os.O_TRUNC
	default:
		flags |= os.O_APPEND
	}

	file, err := os.OpenFile(filename, flags, 0o644)
	if err != nil {
		return err
	}

	for _, msg := range batch {
		data, err := msg.MarshalToBytes()
		if err != nil {
			return err
		}

		_, err = file.Write(append(data, '\n'))
		if err != nil {
			return err
		}
	}

	if err := file.Close(); err != nil {
		return err
	}

	return nil
}

func (o *fileOutput) ProvidesCompression() bool {
	return false
}

func (o *fileOutput) SupportsAggregation() bool {
	return true
}
