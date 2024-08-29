package env

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
)

func init() {
	componentFactories[componentS3] = &s3Factory{}
}

const componentS3 = "s3"

type s3Settings struct {
	ComponentBaseSettings
	ComponentContainerSettings
	Port int `cfg:"port" default:"0"`
}

type s3Factory struct{}

func (f *s3Factory) Detect(config cfg.Config, manager *ComponentsConfigManager) error {
	if !config.IsSet("cloud.aws.s3") {
		return nil
	}

	if !manager.ShouldAutoDetect(componentS3) {
		return nil
	}

	if manager.HasType(componentS3) {
		return nil
	}

	settings := &s3Settings{}
	UnmarshalSettings(config, settings, componentS3, "default")
	settings.Type = componentS3

	if err := manager.Add(settings); err != nil {
		return fmt.Errorf("can not add s3 settings: %w", err)
	}

	return nil
}

func (f *s3Factory) GetSettingsSchema() ComponentBaseSettingsAware {
	return &s3Settings{}
}

func (f *s3Factory) DescribeContainers(settings interface{}) componentContainerDescriptions {
	descriptions := componentContainerDescriptions{
		"main": {
			containerConfig: f.configureContainer(settings),
			healthCheck:     f.healthCheck(),
		},
	}

	return descriptions
}

func (f *s3Factory) configureContainer(settings interface{}) *containerConfig {
	s := settings.(*s3Settings)

	return &containerConfig{
		Repository: s.Image.Repository,
		Tag:        s.Image.Tag,
		Cmd: []string{
			"server",
			"/data",
		},
		Env: []string{
			fmt.Sprintf("MINIO_ACCESS_KEY=%s", DefaultAccessKeyID),
			fmt.Sprintf("MINIO_SECRET_KEY=%s", DefaultSecretAccessKey),
		},
		PortBindings: portBindings{
			"9000/tcp": s.Port,
		},
		ExpireAfter: s.ExpireAfter,
	}
}

func (f *s3Factory) healthCheck() ComponentHealthCheck {
	return func(container *container) error {
		var err error
		var client *s3.Client

		if client, err = f.client(container); err != nil {
			return fmt.Errorf("can not build client: %w", err)
		}
		_, err = client.ListBuckets(context.Background(), &s3.ListBucketsInput{})

		return err
	}
}

func (f *s3Factory) client(container *container) (*s3.Client, error) {
	binding := container.bindings["9000/tcp"]
	address := fmt.Sprintf("http://%s:%s", binding.host, binding.port)

	var err error
	var cfg aws.Config

	if cfg, err = GetDefaultAwsSdkConfig(); err != nil {
		return nil, fmt.Errorf("can't get default aws sdk config: %w", err)
	}

	client := s3.NewFromConfig(cfg, func(options *s3.Options) {
		options.BaseEndpoint = aws.String(address)
	})

	return client, nil
}

func (f *s3Factory) Component(_ cfg.Config, _ log.Logger, containers map[string]*container, settings interface{}) (Component, error) {
	s := settings.(*s3Settings)
	s3Address := fmt.Sprintf("http://%s", containers["main"].bindings["9000/tcp"].getAddress())

	result := &S3Component{
		baseComponent: baseComponent{
			name: s.Name,
		},
		s3Address: s3Address,
	}

	return result, nil
}
