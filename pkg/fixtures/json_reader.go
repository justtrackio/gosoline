package fixtures

import (
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/mon"
)

type jsonFixtureReader struct {}

func NewJsonFixtureReader(cfg cfg.Config, logger mon.Logger) FixtureReader {
	return &jsonFixtureReader{}
}

func (j *jsonFixtureReader) ReadFixtures() (interface{}, error){
	return "",nil
}