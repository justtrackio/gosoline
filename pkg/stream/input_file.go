package stream

import (
	"bufio"
	"bytes"
	"context"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/encoding/json"
	"github.com/applike/gosoline/pkg/mon"
	"os"
)

type FileSettings struct {
	Filename string `cfg:"filename"`
}

type fileInput struct {
	logger   mon.Logger
	settings FileSettings

	channel chan *Message
	stopped bool
}

func NewFileInput(_ cfg.Config, logger mon.Logger, settings FileSettings) Input {
	return NewFileInputWithInterfaces(logger, settings)
}

func NewFileInputWithInterfaces(logger mon.Logger, settings FileSettings) Input {
	return &fileInput{
		logger:   logger,
		settings: settings,
		channel:  make(chan *Message),
	}
}

func (i *fileInput) Data() chan *Message {
	return i.channel
}

func (i *fileInput) Run(ctx context.Context) error {
	defer close(i.channel)

	logger := i.logger.WithContext(ctx).WithFields(mon.Fields{
		"filename": i.settings.Filename,
	})

	file, err := os.Open(i.settings.Filename)

	if err != nil {
		logger.Error("can not open file: %w", err)
		return err
	}

	defer func() {
		err := file.Close()

		if err != nil {
			logger.Error("can not close file: %w", err)
		}
	}()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() && !i.stopped {
		rawMessage := scanner.Bytes()

		// skip over empty lines
		rawMessage = bytes.TrimSpace(rawMessage)
		if len(rawMessage) == 0 {
			continue
		}

		msg := &Message{}
		err = json.Unmarshal(rawMessage, msg)

		if err != nil {
			logger.Error("could not unmarshal message: %w", err)
			continue
		}

		i.channel <- msg
	}

	return nil
}

func (i *fileInput) Stop() {
	i.stopped = true
}
