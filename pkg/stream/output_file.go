package stream

import (
	"context"
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/mon"
	"github.com/hashicorp/go-multierror"
	"os"
	"sync"
)

type FileOutputSettings struct {
	Filename string `cfg:"filename"`
	Append   bool   `cfg:"append"`
}

type fileOutput struct {
	logger   mon.Logger
	settings FileOutputSettings
	lck      sync.Mutex
}

func NewFileOutput(_ cfg.Config, logger mon.Logger, settings *FileOutputSettings) Output {
	return &fileOutput{
		logger:   logger,
		settings: *settings,
		lck:      sync.Mutex{},
	}
}

func (o *fileOutput) WriteOne(ctx context.Context, msg WritableMessage) error {
	return o.Write(ctx, []WritableMessage{msg})
}

func (o *fileOutput) Write(_ context.Context, batch []WritableMessage) (writeErr error) {
	o.lck.Lock()
	defer o.lck.Unlock()

	flags := os.O_CREATE | os.O_WRONLY
	if o.settings.Append {
		flags = flags | os.O_APPEND
	} else {
		flags = flags | os.O_TRUNC
	}

	file, err := os.OpenFile(o.settings.Filename, flags, 0644)

	if err != nil {
		return fmt.Errorf("failed to open file %s: %w", o.settings.Filename, err)
	}

	// from now on, always append, otherwise we will truncate what whe just wrote on the next call
	o.settings.Append = true

	defer func() {
		if err := file.Close(); err != nil {
			err = fmt.Errorf("failed to close file %s: %w", o.settings.Filename, err)

			if writeErr != nil {
				writeErr = multierror.Append(writeErr, err)
			} else {
				writeErr = err
			}
		}
	}()

	for _, msg := range batch {
		data, err := msg.MarshalToBytes()

		if err != nil {
			return fmt.Errorf("failed to marshal message: %w", err)
		}

		_, err = file.Write(append(data, '\n'))

		if err != nil {
			return fmt.Errorf("failed to write message to file %s: %w", o.settings.Filename, err)
		}
	}

	return nil
}
