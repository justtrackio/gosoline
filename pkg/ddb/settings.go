package ddb

import (
	"github.com/applike/gosoline/pkg/cloud"
	"github.com/applike/gosoline/pkg/exec"
	"github.com/applike/gosoline/pkg/mdl"
)

const defaultMaxWaitSeconds = 60

type Settings struct {
	ModelId        mdl.ModelId
	NamingStrategy NamingFactory
	AutoCreate     bool
	Client         cloud.ClientSettings
	Backoff        exec.BackoffSettings
	Main           MainSettings
	Local          []LocalSettings
	Global         []GlobalSettings
}

func (s *Settings) WithModelId(modelId mdl.ModelId) *Settings {
	s.ModelId = modelId

	return s
}

type MainSettings struct {
	Model              interface{}
	StreamView         string
	ReadCapacityUnits  int64
	WriteCapacityUnits int64
}

type LocalSettings struct {
	Name  string
	Model interface{}
}

type GlobalSettings struct {
	Name               string
	Model              interface{}
	ReadCapacityUnits  int64
	WriteCapacityUnits int64
}

type SimpleSettings struct {
	ModelId            mdl.ModelId
	AutoCreate         bool
	Model              interface{}
	StreamView         string
	ReadCapacityUnits  int64
	WriteCapacityUnits int64
}
