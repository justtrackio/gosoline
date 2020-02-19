package test

import (
	"fmt"
	"github.com/aws/aws-sdk-go/service/kinesis"
	"log"
)

type kinesisConfig struct {
	Debug bool   `mapstructure:"debug"`
	Host  string `mapstructure:"host"`
	Port  int    `mapstructure:"port"`
}

var kinesisConfigs map[string]*kinesisConfig
var kinesisClients simpleCache

func init() {
	kinesisConfigs = map[string]*kinesisConfig{}
	kinesisClients = simpleCache{}
}

func ProvideKinesisClient(name string) *kinesis.Kinesis {
	return kinesisClients.New(name, func() interface{} {

		sess, err := getSession(kinesisConfigs[name].Host, kinesisConfigs[name].Port)

		if err != nil {
			logErr(err, "could not create kinesis client: %s")
		}

		return kinesis.New(sess)

	}).(*kinesis.Kinesis)
}

func runKinesis(name string, config configInput) {
	wait.Add(1)
	go doRunKinesis(name, config)
}

func doRunKinesis(name string, configMap configInput) {
	defer wait.Done()
	defer log.Printf("%s component of type %s is ready", name, "kinesis")

	localConfig := &kinesisConfig{}
	unmarshalConfig(configMap, localConfig)
	kinesisConfigs[name] = localConfig

	containerName := fmt.Sprintf("gosoline_test_kinesis_%s", name)

	runContainer(containerName, ContainerConfig{
		Repository: "localstack/localstack",
		Tag:        "0.10.7",
		Env: []string{
			"SERVICES=kinesis",
		},
		PortBindings: PortBinding{
			"4568/tcp": fmt.Sprint(localConfig.Port),
		},
		HealthCheck: localstackHealthCheck(containerName),
		PrintLogs:   localConfig.Debug,
	})
}
