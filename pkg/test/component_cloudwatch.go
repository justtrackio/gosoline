package test

import (
	"fmt"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"log"
	"sync"
)

type cloudwatchConfig struct {
	Port int `mapstructure:"port"`
}

var cloudwatchClients map[string]*cloudwatch.CloudWatch
var cloudwatchConfigs map[string]*cloudwatchConfig
var cloudwatchLck sync.Mutex

func init() {
	cloudwatchConfigs = map[string]*cloudwatchConfig{}
	cloudwatchClients = map[string]*cloudwatch.CloudWatch{}
}

func ProvideCloudwatchClient(name string) *cloudwatch.CloudWatch {
	cloudwatchLck.Lock()
	defer cloudwatchLck.Unlock()

	_, ok := cloudwatchClients[name]
	if ok {
		return cloudwatchClients[name]
	}

	sess, err := getSession(cloudwatchConfigs[name].Port)

	if err != nil {
		logErr(err, "could not create cloudwatch client: %s")
	}

	cloudwatchClients[name] = cloudwatch.New(sess)

	return cloudwatchClients[name]
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

	runContainer("gosoline-test-cloudwatch", ContainerConfig{
		Repository: "localstack/localstack",
		Tag:        "0.10.3",
		Env: []string{
			"SERVICES=cloudwatch",
		},
		PortBindings: PortBinding{
			"4582/tcp": fmt.Sprint(localConfig.Port),
		},
		HealthCheck: func() error {
			cloudwatchClient := ProvideCloudwatchClient(name)
			_, err := cloudwatchClient.ListDashboards(&cloudwatch.ListDashboardsInput{})

			return err
		},
	})
}
