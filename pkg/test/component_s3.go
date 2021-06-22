package test

import (
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/log"
	"github.com/aws/aws-sdk-go/service/s3"
)

const componentS3 = "s3"

type s3Settings struct {
	*mockSettings
	Port int `cfg:"port" default:"0"`
}

type s3Component struct {
	mockComponentBase
	settings *s3Settings
	clients  *simpleCache
}

func (k *s3Component) Boot(config cfg.Config, logger log.Logger, runner *dockerRunnerLegacy, settings *mockSettings, name string) {
	k.logger = logger
	k.name = name
	k.runner = runner
	k.clients = &simpleCache{}
	k.settings = &s3Settings{
		mockSettings: settings,
	}
	key := fmt.Sprintf("mocks.%s", name)
	config.UnmarshalKey(key, k.settings)
}

func (k *s3Component) Start() error {
	containerName := fmt.Sprintf("gosoline_test_s3_%s", k.name)

	return k.runner.Run(containerName, &containerConfigLegacy{
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
			"9000/tcp": fmt.Sprint(k.settings.Port),
		},
		PortMappings: portMappingLegacy{
			"9000/tcp": &k.settings.Port,
		},
		HostMapping: hostMappingLegacy{
			dialPort: &k.settings.Port,
			setHost:  &k.settings.Host,
		},
		HealthCheck: func() error {
			_, err := k.provideS3Client().ListBuckets(&s3.ListBucketsInput{})

			return err
		},
		PrintLogs:   k.settings.Debug,
		ExpireAfter: k.settings.ExpireAfter,
	})
}

func (k *s3Component) provideS3Client() *s3.S3 {
	return k.clients.New(k.name, func() interface{} {
		sess := getAwsSession(k.settings.Host, k.settings.Port)

		return s3.New(sess)
	}).(*s3.S3)
}
