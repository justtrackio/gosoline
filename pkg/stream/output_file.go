package stream

import (
	"context"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/log"
	"os"
)

type FileOutputSettings struct {
	Filename string `cfg:"filename"`
	Append   bool   `cfg:"append"`
}

type fileOutput struct {
	logger   log.Logger
	settings *FileOutputSettings
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
	flags := os.O_CREATE | os.O_WRONLY
	if o.settings.Append {
		flags = flags | os.O_APPEND
	} else {
		flags = flags | os.O_TRUNC
	}

	file, err := os.OpenFile(o.settings.Filename, flags, 0644)

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

	if err = file.Close(); err != nil {
		return err
	}

	return nil
}
