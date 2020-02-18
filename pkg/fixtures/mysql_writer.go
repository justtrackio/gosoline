package fixtures

import (
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/mon"
)

type mySqlFixtureWriter struct {
	config cfg.Config
	logger mon.Logger
	reader FixtureReader
}

func NewMySqlFixtureWriter(cfg cfg.Config, logger mon.Logger, reader FixtureReader) FixtureWriter {
	return &mySqlFixtureWriter{
		config: cfg,
		logger: logger,
		reader: reader,
	}
}

func (m *mySqlFixtureWriter) WriteFixtures() error {
		_, err := m.reader.ReadFixtures()

		return err
}