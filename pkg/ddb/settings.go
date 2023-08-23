package ddb

import (
	"math"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/justtrackio/gosoline/pkg/mdl"
)

const defaultMaxWaitSeconds = 60

type Settings struct {
	ModelId             mdl.ModelId
	TableNamingSettings TableNamingSettings
	AutoCreate          bool
	DisableTracing      bool
	ClientName          string
	Main                MainSettings
	Local               []LocalSettings
	Global              []GlobalSettings
}

type MainSettings struct {
	Model              interface{}
	StreamView         types.StreamViewType
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

func sanitizeSettings(settings *Settings) {
	if len(settings.ClientName) == 0 {
		settings.ClientName = "default"
	}

	settings.Main.ReadCapacityUnits = int64(math.Max(1, float64(settings.Main.ReadCapacityUnits)))
	settings.Main.WriteCapacityUnits = int64(math.Max(1, float64(settings.Main.WriteCapacityUnits)))

	for _, global := range settings.Global {
		global.ReadCapacityUnits = int64(math.Max(1, float64(global.ReadCapacityUnits)))
		global.WriteCapacityUnits = int64(math.Max(1, float64(global.WriteCapacityUnits)))
	}
}
