package test

import (
	"fmt"

	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
)

const componentS3 = "s3"

type s3Settings struct {
	*mockSettings
}

type s3Component struct {
	mockComponentBase
	settings *s3Settings
	clients  *simpleCache
}

func (s *s3Component) Boot(config cfg.Config, logger log.Logger, runner *dockerRunnerLegacy, settings *mockSettings, name string) {
	s.logger = logger
	s.name = name
	s.runner = runner
	s.clients = &simpleCache{}
	s.settings = &s3Settings{
		mockSettings: settings,
	}
	key := fmt.Sprintf("mocks.%s", name)
	config.UnmarshalKey(key, s.settings)
}

func (s *s3Component) getContainerConfig() *containerConfigLegacy {
	return &containerConfigLegacy{
		Repository: "minio/minio",
		Cmd: []string{
			"server",
			"/data",
		},
		Tag: "RELEASE.2020-12-03T05-49-24Z",
		Env: []string{
			"MINIO_ACCESS_KEY=gosoline",
			"MINIO_SECRET_KEY=gosoline",
		},
		PortBindings: portBindingLegacy{
			"9000/tcp": fmt.Sprint(s.settings.Port),
		},
		PortMappings: portMappingLegacy{
			"9000/tcp": &s.settings.Port,
		},
		HostMapping: hostMappingLegacy{
			dialPort: &s.settings.Port,
			setHost:  &s.settings.Host,
		},
		HealthCheck: func() error {
			_, err := s.provideS3Client().ListBuckets(&s3.ListBucketsInput{})

			return err
		},
		PrintLogs:   s.settings.Debug,
		ExpireAfter: s.settings.ExpireAfter,
	}
}

func (s *s3Component) PullContainerImage() error {
	containerName := fmt.Sprintf("gosoline_test_s3_%s", s.name)
	containerConfig := s.getContainerConfig()

	return s.runner.PullContainerImage(containerName, containerConfig)
}

func (s *s3Component) Start() error {
	containerName := fmt.Sprintf("gosoline_test_s3_%s", s.name)
	containerConfig := s.getContainerConfig()

	return s.runner.Run(containerName, containerConfig)
}

func (s *s3Component) provideS3Client() *s3.S3 {
	return s.clients.New(s.name, func() interface{} {
		sess := getAwsSession(s.settings.Host, s.settings.Port)

		return s3.New(sess)
	}).(*s3.S3)
}
