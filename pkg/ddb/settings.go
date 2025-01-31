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
	DisableTracing      bool
	ClientName          string
	Main                MainSettings
	Local               []LocalSettings
	Global              []GlobalSettings
}

type MainSettings struct {
	Model              any
	StreamView         types.StreamViewType
	ReadCapacityUnits  int64
	WriteCapacityUnits int64
}

type LocalSettings struct {
	Name  string
	Model any
}

type GlobalSettings struct {
	Name               string
	Model              any
	ReadCapacityUnits  int64
	WriteCapacityUnits int64
}

type SimpleSettings struct {
	ModelId            mdl.ModelId
	AutoCreate         bool
	Model              any
	StreamView         string
	ReadCapacityUnits  int64
	WriteCapacityUnits int64
}

func sanitizeSettings(settings *Settings) {
	if settings.ClientName == "" {
		settings.ClientName = "default"
	}

	settings.Main.ReadCapacityUnits = int64(math.Max(1, float64(settings.Main.ReadCapacityUnits)))
	settings.Main.WriteCapacityUnits = int64(math.Max(1, float64(settings.Main.WriteCapacityUnits)))

	for i := range settings.Global {
		// work on a reference to ensure our update is correctly propagated
		global := &settings.Global[i]
		global.ReadCapacityUnits = int64(math.Max(1, float64(global.ReadCapacityUnits)))
		global.WriteCapacityUnits = int64(math.Max(1, float64(global.WriteCapacityUnits)))
	}
}
