package blob

import (
	"fmt"

	"github.com/justtrackio/gosoline/pkg/cfg"
	gosoS3 "github.com/justtrackio/gosoline/pkg/cloud/aws/s3"
)

type BatchRunnerSettings struct {
	ClientName        string `cfg:"client_name"         default:"default"`
	CopyRunnerCount   int    `cfg:"copy_runner_count"   default:"10"`
	DeleteRunnerCount int    `cfg:"delete_runner_count" default:"10"`
	ReaderRunnerCount int    `cfg:"reader_runner_count" default:"10"`
	WriterRunnerCount int    `cfg:"writer_runner_count" default:"10"`
}

type Settings struct {
	cfg.AppId
	Bucket     string `cfg:"bucket"`
	Region     string `cfg:"region"`
	ClientName string `cfg:"client_name" default:"default"`
	Prefix     string `cfg:"prefix"`
}

func getStoreSettings(config cfg.Config, name string) *Settings {
	settings := &Settings{}
	key := fmt.Sprintf("blob.%s", name)
	config.UnmarshalKey(key, settings)
	settings.AppId.PadFromConfig(config)

	if settings.Bucket == "" {
		settings.Bucket = fmt.Sprintf("%s-%s-%s", settings.Project, settings.Environment, settings.Family)
	}

	if settings.Region == "" {
		s3ClientConfig := gosoS3.GetClientConfig(config, settings.ClientName)
		settings.Region = s3ClientConfig.Settings.Region
	}

	return settings
}
