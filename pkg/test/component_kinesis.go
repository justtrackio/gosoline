package test

import (
	"fmt"
	"github.com/applike/gosoline/pkg/mdl"
	"github.com/aws/aws-sdk-go/service/kinesis"
	"log"
	"sync"
	"time"
)

type kinesisConfig struct {
	Host string `mapstructure:"host"`
	Port int    `mapstructure:"port"`
}

var kinesisConfigs map[string]*kinesisConfig
var kinesisClients map[string]*kinesis.Kinesis
var kinesisLck sync.Mutex

func init() {
	kinesisConfigs = map[string]*kinesisConfig{}
	kinesisClients = map[string]*kinesis.Kinesis{}
}

func ProvideKinesisClient(name string) *kinesis.Kinesis {
	kinesisLck.Lock()
	defer kinesisLck.Unlock()

	_, ok := kinesisClients[name]
	if ok {
		return kinesisClients[name]
	}

	sess, err := getSession(kinesisConfigs[name].Host, kinesisConfigs[name].Port)

	if err != nil {
		logErr(err, "could not create kinesis client: %s")
	}

	kinesisClients[name] = kinesis.New(sess)

	return kinesisClients[name]
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

	runContainer("gosoline_test_kinesis", ContainerConfig{
		Repository: "localstack/localstack",
		Tag:        "0.10.3",
		Env: []string{
			"SERVICES=kinesis",
		},
		PortBindings: PortBinding{
			"4568/tcp": fmt.Sprint(localConfig.Port),
		},
		HealthCheck: func() error {
			kinesisClient := ProvideKinesisClient(name)
			streamName := "healthcheck"

			_, err := kinesisClient.CreateStream(&kinesis.CreateStreamInput{
				ShardCount: mdl.Int64(1),
				StreamName: mdl.String(streamName),
			})

			if err != nil {
				return err
			}

			listStreams, err := kinesisClient.ListStreams(&kinesis.ListStreamsInput{})

			if err != nil {
				return err
			}

			if len(listStreams.StreamNames) != 1 {
				return fmt.Errorf("stream list should contain exactly 1 entry, but contained %d", len(listStreams.StreamNames))
			}

			_, err = kinesisClient.DeleteStream(&kinesis.DeleteStreamInput{StreamName: mdl.String(streamName)})

			if err != nil {
				return err
			}

			// wait for stream to be really deleted (race condition)
			for {
				listStreams, err := kinesisClient.ListStreams(&kinesis.ListStreamsInput{})

				if err != nil {
					return err
				}

				if len(listStreams.StreamNames) == 0 {
					return nil
				}

				time.Sleep(50 * time.Millisecond)
			}
		},
	})
}
