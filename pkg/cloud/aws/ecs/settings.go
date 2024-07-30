package ecs

import (
	"github.com/aws/aws-sdk-go-v2/aws/retry"
	awsCfg "github.com/aws/aws-sdk-go-v2/config"
	gosoAws "github.com/justtrackio/gosoline/pkg/cloud/aws"
)

type ClientSettings struct {
	gosoAws.ClientSettings
}

type ClientConfig struct {
	Settings    ClientSettings
	LoadOptions []func(options *awsCfg.LoadOptions) error
}

func (c ClientConfig) GetSettings() gosoAws.ClientSettings {
	return c.Settings.ClientSettings
}

func (c ClientConfig) GetLoadOptions() []func(options *awsCfg.LoadOptions) error {
	return c.LoadOptions
}

func (c ClientConfig) GetRetryOptions() []func(*retry.StandardOptions) {
	return nil
}

type ClientOption func(cfg *ClientConfig)
