package test

import (
	"fmt"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"log"
)

type cloudwatchConfig struct {
	Debug bool   `mapstructure:"debug"`
	Host  string `mapstructure:"host"`
	Port  int    `mapstructure:"port"`
}

var cloudwatchConfigs map[string]*cloudwatchConfig
var cloudwatchClients simpleCache

func init() {
	cloudwatchConfigs = map[string]*cloudwatchConfig{}
	cloudwatchClients = simpleCache{}
}

func ProvideCloudwatchClient(name string) *cloudwatch.CloudWatch {
	return cloudwatchClients.New(name, func() interface{} {
		sess, err := getSession(cloudwatchConfigs[name].Host, cloudwatchConfigs[name].Port)

		if err != nil {
			logErr(err, fmt.Sprintf("could not create cloudwatch client: %s", name))
		}

		return cloudwatch.New(sess)
	}).(*cloudwatch.CloudWatch)
}

func runCloudwatch(name string, config configInput) {
	wait.Add(1)
	go doRunCloudwatch(name, config)
}

func doRunCloudwatch(name string, configMap configInput) {
	defer wait.Done()
	defer log.Printf("%s component of type %s is ready", name, "cloudwatch")

	localConfig := &cloudwatchConfig{}
	unmarshalConfig(configMap, localConfig)
	cloudwatchConfigs[name] = localConfig

	containerName := fmt.Sprintf("gosoline_test_cloudwatch_%s", name)

	runContainer(containerName, ContainerConfig{
		Repository: "localstack/localstack",
		Tag:        "0.10.7",
		Env: []string{
			"SERVICES=cloudwatch",
		},
		PortBindings: PortBinding{
			"4582/tcp": fmt.Sprint(localConfig.Port),
		},
		HealthCheck: localstackHealthCheck(containerName),
		PrintLogs:   localConfig.Debug,
	})
}
