package stream

import (
	"context"
	"encoding/json"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/mon"
	"os"
)

type fileOutput struct {
	logger   mon.Logger
	settings FileSettings
}

func NewFileOutput(config cfg.Config, logger mon.Logger, settings FileSettings) Output {
	return &fileOutput{
		logger:   logger,
		settings: settings,
	}
}

func (o *fileOutput) WriteOne(ctx context.Context, msg *Message) error {
	return o.Write(ctx, []*Message{msg})
}

func (o *fileOutput) Write(ctx context.Context, batch []*Message) error {
	file, err := os.OpenFile(o.settings.Filename, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)

	if err != nil {
		return err
	}

	for _, msg := range batch {
		data, err := json.Marshal(*msg)

		if err != nil {
			return err
		}

		_, err = file.Write(data)

		if err != nil {
			return err
		}

		_, err = file.Write([]byte{'\n'})

		if err != nil {
			return err
		}
	}

	if err = file.Close(); err != nil {
		return err
	}

	return nil
}
