package ddb

import (
	"github.com/applike/gosoline/pkg/cloud"
	"github.com/applike/gosoline/pkg/mdl"
)

const defaultMaxWaitSeconds = 60

type Settings struct {
	ModelId    mdl.ModelId
	AutoCreate bool
	Client     cloud.ClientSettings
	Backoff    cloud.BackoffSettings
	Main       MainSettings
	Local      []LocalSettings
	Global     []GlobalSettings
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
