package env

import (
	"context"
	"fmt"

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

type s3Factory struct{}

func (f *s3Factory) Detect(config cfg.Config, manager *ComponentsConfigManager) error {
	if !config.IsSet("cloud.aws.s3") {
		return nil
	}

	if manager.HasType(componentS3) {
		return nil
	}

	settings := &s3Settings{}
	config.UnmarshalDefaults(settings)

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
		Repository: "minio/minio",
		Tag:        "RELEASE.2021-09-18T18-09-59Z",
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
		s3Client := f.client(container)
		_, err := s3Client.ListBuckets(context.Background(), &s3.ListBucketsInput{})

		return err
	}
}

func (f *s3Factory) client(container *container) *s3.Client {
	binding := container.bindings["9000/tcp"]
	address := fmt.Sprintf("http://%s:%s", binding.host, binding.port)

	awsCfg := aws.Config{
		EndpointResolverWithOptions: gosoAws.EndpointResolver(address),
		Region:                      "eu-central-1",
		Credentials:                 GetDefaultStaticCredentials(),
	}

	return s3.NewFromConfig(awsCfg)
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
