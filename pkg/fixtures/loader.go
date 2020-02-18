package fixtures

import (
	"context"
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/kernel"
	"github.com/applike/gosoline/pkg/mon"
)

type FixtureLoader struct {
	kernel.BackgroundModule
	logger       mon.Logger
	Writers []FixtureWriter
	Settings []FixtureLoaderSettings `cfg:"fixture_loader"`
}

func NewFixtureLoader() *FixtureLoader {
	return &FixtureLoader{}
}

func (f *FixtureLoader) Boot(config cfg.Config, logger mon.Logger) error {
	f.logger = logger

	if !config.IsSet("fixture_loader"){
		f.logger.Info("fixture loader module not configured")
		return nil
	}

	var s []FixtureLoaderSettings
	config.UnmarshalKey("fixture_loader", &s)
	f.Settings = s

	err := f.BootFixtureWriters(config, logger)

	if err != nil {
		return err
	}

	for _, writer := range f.Writers {
		err := writer.WriteFixtures()

		if err != nil {
			f.logger.Error(err, "error occurred during fixture loading")
			return err
		}
	}

	return nil
}

func (f *FixtureLoader) BootFixtureWriters(config cfg.Config, logger mon.Logger) error {
	for _,s := range f.Settings {
		if !s.Enable {
			f.logger.Info(fmt.Sprintf("fixture writer for input %s and output %s not enabled", s.Input.Encoding, s.Output))
			continue
		}

		readerFactory, ok := fixtureReaders[s.Input.Encoding]

		if !ok {
			f.logger.Warn("no reader implemented for input encoding ", s.Input.Encoding)
			return fmt.Errorf("no reader implemented for input encoding %s", s.Input.Encoding)
		}

		fixtureWriterFactory, ok := fixtureWriters[s.Output]

		if !ok {
			f.logger.Warn("no writer existing for output type ", s.Output)
			return fmt.Errorf("no writer existing for output type %s", s.Output)
		}

		reader := readerFactory(config, logger)
		f.Writers = append(f.Writers, fixtureWriterFactory(config, logger, reader))
	}

	return nil
}

func (f *FixtureLoader) Run(ctx context.Context) error {
	// do nothing: fixtures are loaded during boot
	return nil
}

