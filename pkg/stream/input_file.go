package stream

import (
	"bufio"
	"encoding/json"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/mon"
	"os"
)

type FileSettings struct {
	Filename string `mapstructure:"filename"`
	Blocking bool
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

func (i *fileInput) Run() error {
	defer func() {
		if !i.settings.Blocking {
			close(i.channel)
		}
	}()

	file, err := os.Open(i.settings.Filename)

	if err != nil {
		i.logger.Error(err, "can not open file")
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
			i.logger.Error(err, "could not unmarshal message")
			continue
		}

		i.channel <- &msg
	}

	return nil
}

func (i *fileInput) Stop() {
	i.stopped = true
}
