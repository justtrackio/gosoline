package env

import (
	"context"
	"fmt"
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/justtrackio/gosoline/pkg/cfg"
	gosoAws "github.com/justtrackio/gosoline/pkg/cloud/aws"
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

type s3Factory struct {
	lck     sync.Mutex
	clients map[string]*s3.Client
}

func (f *s3Factory) Detect(config cfg.Config, manager *ComponentsConfigManager) error {
	if !config.IsSet("cloud.aws.s3") {
		return nil
	}

	if !manager.ShouldAutoDetect(componentS3) {
		return nil
	}

	if has, err := manager.HasType(componentS3); err != nil {
		return fmt.Errorf("failed to check if component exists: %w", err)
	} else if has {
		return nil
	}

	settings := &s3Settings{}
	if err := UnmarshalSettings(config, settings, componentS3, "default"); err != nil {
		return fmt.Errorf("can not unmarshal S3 settings: %w", err)
	}
	settings.Type = componentS3

	if err := manager.Add(settings); err != nil {
		return fmt.Errorf("can not add s3 settings: %w", err)
	}

	return nil
}

func (f *s3Factory) GetSettingsSchema() ComponentBaseSettingsAware {
	return &s3Settings{}
}

func (f *s3Factory) DescribeContainers(settings any) ComponentContainerDescriptions {
	descriptions := ComponentContainerDescriptions{
		"main": {
			ContainerConfig: f.configureContainer(settings),
			HealthCheck:     f.healthCheck(),
		},
	}

	return descriptions
}

func (f *s3Factory) configureContainer(settings any) *ContainerConfig {
	s := settings.(*s3Settings)

	return &ContainerConfig{
		Auth:       s.Image.Auth,
		Repository: s.Image.Repository,
		Tag:        s.Image.Tag,
		Cmd: []string{
			"server",
			"/data",
		},
		Env: map[string]string{
			"MINIO_ACCESS_KEY": DefaultAccessKeyID,
			"MINIO_SECRET_KEY": DefaultSecretAccessKey,
		},
		PortBindings: PortBindings{
			"main": {
				ContainerPort: 9000,
				HostPort:      s.Port,
				Protocol:      "tcp",
			},
		},
	}
}

func (f *s3Factory) healthCheck() ComponentHealthCheck {
	return func(container *Container) error {
		var err error
		var client *s3.Client

		if client, err = f.client(container); err != nil {
			return fmt.Errorf("can not build client: %w", err)
		}
		_, err = client.ListBuckets(context.Background(), &s3.ListBucketsInput{})

		return err
	}
}

func (f *s3Factory) client(container *Container) (*s3.Client, error) {
	binding := container.bindings["main"]
	address := fmt.Sprintf("http://%s:%s", binding.host, binding.port)

	f.lck.Lock()
	defer f.lck.Unlock()

	if f.clients == nil {
		f.clients = make(map[string]*s3.Client)
	}

	if _, ok := f.clients[address]; !ok {
		var err error
		var cfg aws.Config

		if cfg, err = GetDefaultAwsSdkConfig(); err != nil {
			return nil, fmt.Errorf("can't get default aws sdk config: %w", err)
		}

		f.clients[address] = s3.NewFromConfig(cfg, func(options *s3.Options) {
			options.BaseEndpoint = gosoAws.NilIfEmpty(address)
		})
	}

	return f.clients[address], nil
}

func (f *s3Factory) Component(_ cfg.Config, _ log.Logger, containers map[string]*Container, settings any) (Component, error) {
	s := settings.(*s3Settings)
	s3Address := fmt.Sprintf("http://%s", containers["main"].bindings["main"].getAddress())

	result := &S3Component{
		baseComponent: baseComponent{
			name: s.Name,
		},
		s3Address: s3Address,
	}

	return result, nil
}
