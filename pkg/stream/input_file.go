package stream

import (
	"bufio"
	"context"
	"os"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/encoding/json"
	"github.com/justtrackio/gosoline/pkg/log"
)

type FileSettings struct {
	Filename string `cfg:"filename"`
	Blocking bool   `cfg:"blocking"`
}

type fileInput struct {
	logger   log.Logger
	settings FileSettings

	channel chan *Message
	stopped bool
}

func NewFileInput(_ cfg.Config, logger log.Logger, settings FileSettings) Input {
	return NewFileInputWithInterfaces(logger, settings)
}

func NewFileInputWithInterfaces(logger log.Logger, settings FileSettings) Input {
	return &fileInput{
		logger:   logger,
		settings: settings,
		channel:  make(chan *Message),
	}
}

func (i *fileInput) Data() <-chan *Message {
	return i.channel
}

func (i *fileInput) Run(_ context.Context) error {
	defer func() {
		if !i.settings.Blocking {
			close(i.channel)
		}
	}()

	file, err := os.Open(i.settings.Filename)
	if err != nil {
		i.logger.Error("can not open file: %w", err)

		return err
	}

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		if i.stopped {
			break
		}

		rawMessage := scanner.Text()

		msg := Message{}
		err = json.Unmarshal([]byte(rawMessage), &msg)
		if err != nil {
			i.logger.Error("could not unmarshal message: %w", err)

			continue
		}

		i.channel <- &msg
	}

	return nil
}

func (i *fileInput) Stop() {
	i.stopped = true
}

func (i *fileInput) IsHealthy() bool {
	return true
}
