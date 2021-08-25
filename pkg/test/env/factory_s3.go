package env

import (
	"fmt"
	"os"

	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/log"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
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
	if !config.IsSet("aws_s3_endpoint") {
		return nil
	}

	if manager.HasType(componentS3) {
		return nil
	}

	_ = os.Setenv("AWS_ACCESS_KEY_ID", "gosoline")
	_ = os.Setenv("AWS_SECRET_ACCESS_KEY", "gosoline")

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
		Tag:        "RELEASE.2020-12-03T05-49-24Z",
		Cmd: []string{
			"server",
			"/data",
		},
		Env: []string{
			"MINIO_ACCESS_KEY=gosoline",
			"MINIO_SECRET_KEY=gosoline",
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
		_, err := s3Client.ListBuckets(&s3.ListBucketsInput{})

		return err
	}
}

func (f *s3Factory) client(container *container) *s3.S3 {
	binding := container.bindings["9000/tcp"]
	address := fmt.Sprintf("http://%s:%s", binding.host, binding.port)

	sess := session.Must(session.NewSession(&aws.Config{
		Endpoint:   aws.String(address),
		Region:     aws.String("eu-central-1"),
		MaxRetries: aws.Int(0),
	}))

	return s3.New(sess)
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
