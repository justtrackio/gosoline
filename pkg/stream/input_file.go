package stream

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"sync"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/conc"
	"github.com/justtrackio/gosoline/pkg/encoding/json"
	"github.com/justtrackio/gosoline/pkg/log"
)

type FileSettings struct {
	Filename    string `cfg:"filename"`
	RunnerCount int    `cfg:"runner_count" default:"1" validate:"min=1"`
}

type fileInput struct {
	logger   log.Logger
	settings FileSettings

	stopped conc.SignalOnce
}

var _ Input = &fileInput{}

func NewFileInput(_ cfg.Config, logger log.Logger, settings FileSettings) Input {
	return NewFileInputWithInterfaces(logger, settings)
}

func NewFileInputWithInterfaces(logger log.Logger, settings FileSettings) Input {
	if settings.RunnerCount <= 0 {
		settings.RunnerCount = 1
	}

	return &fileInput{
		logger:   logger,
		settings: settings,
		stopped:  conc.NewSignalOnce(),
	}
}

func (i *fileInput) Run(ctx context.Context, process InputProcess) (err error) {
	var file *os.File
	var workers sync.WaitGroup

	messages := make(chan *Message)

	if file, err = os.Open(i.settings.Filename); err != nil {
		i.logger.Error(ctx, "can not open file: %w", err)

		return err
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			err = errors.Join(err, fmt.Errorf("can not close input file: %w", closeErr))
		}
	}()

	for range i.settings.RunnerCount {
		workers.Go(func() {
			for msg := range messages {
				process(ctx, msg)
			}
		})
	}
	defer func() {
		close(messages)
		workers.Wait()
	}()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return nil
		case <-i.stopped.Channel():
			return nil
		default:
		}

		rawMessage := scanner.Text()

		msg := Message{}
		err = json.Unmarshal([]byte(rawMessage), &msg)
		if err != nil {
			i.logger.Error(ctx, "could not unmarshal message: %w", err)

			continue
		}

		select {
		case <-ctx.Done():
			return nil
		case <-i.stopped.Channel():
			return nil
		case messages <- &msg:
		}
	}

	if err := scanner.Err(); err != nil {
		i.logger.Error(ctx, "could not read file: %w", err)

		return err
	}

	return nil
}

func (i *fileInput) Stop(_ context.Context) {
	i.stopped.Signal()
}

func (i *fileInput) IsHealthy() bool {
	return true
}
