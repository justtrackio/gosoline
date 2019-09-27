package test

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/endpoints"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/aws/aws-sdk-go/service/kinesis"
	"github.com/aws/aws-sdk-go/service/sns"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/hashicorp/go-multierror"
	"log"
	"net/http"
	"sync"
	"time"
)

type localstackConfig struct {
	CloudwatchPort int    `mapstructure:"cloudwatch_port"`
	KinesisPort    int    `mapstructure:"kinesis_port"`
	Services       string `mapstructure:"services"`
	SnsPort        int    `mapstructure:"sns_port"`
	SqsPort        int    `mapstructure:"sqs_port"`
}

var cloudwatchClients map[string]*cloudwatch.CloudWatch
var kinesisClients map[string]*kinesis.Kinesis
var snsClients map[string]*sns.SNS
var sqsClients map[string]*sqs.SQS
var configs map[string]*localstackConfig
var lck sync.Mutex

func init() {
	configs = map[string]*localstackConfig{}

	cloudwatchClients = map[string]*cloudwatch.CloudWatch{}
	kinesisClients = map[string]*kinesis.Kinesis{}
	snsClients = map[string]*sns.SNS{}
	sqsClients = map[string]*sqs.SQS{}
}

func ProvideCloudwatchClient(name string) *cloudwatch.CloudWatch {
	lck.Lock()
	defer lck.Unlock()

	_, ok := cloudwatchClients[name]
	if ok {
		return cloudwatchClients[name]
	}

	sess, err := getSession(configs[name].CloudwatchPort)

	if err != nil {
		logErr(err, "could not create cloudwatch client: %s")
	}

	cloudwatchClients[name] = cloudwatch.New(sess)

	return cloudwatchClients[name]
}

func ProvideSnsClient(name string) *sns.SNS {
	lck.Lock()
	defer lck.Unlock()

	_, ok := snsClients[name]
	if ok {
		return snsClients[name]
	}

	sess, err := getSession(configs[name].SnsPort)

	if err != nil {
		logErr(err, "could not create sns client: %s")
	}

	snsClients[name] = sns.New(sess)

	return snsClients[name]
}

func getSession(port int) (*session.Session, error) {
	host := fmt.Sprintf("http://localhost:%d", port)

	config := &aws.Config{
		Region:   aws.String(endpoints.EuCentral1RegionID),
		Endpoint: aws.String(host),
		HTTPClient: &http.Client{
			Timeout: 1 * time.Minute,
		},
	}

	return session.NewSession(config)
}

func ProvideSqsClient(name string) *sqs.SQS {
	lck.Lock()
	defer lck.Unlock()

	_, ok := sqsClients[name]
	if ok {
		return sqsClients[name]
	}

	sess, err := getSession(configs[name].SqsPort)

	if err != nil {
		logErr(err, "could not create sqs client: %s")
	}

	sqsClients[name] = sqs.New(sess)

	return sqsClients[name]
}

func ProvideKinesisClient(name string) *kinesis.Kinesis {
	lck.Lock()
	defer lck.Unlock()

	_, ok := kinesisClients[name]
	if ok {
		return kinesisClients[name]
	}

	sess, err := getSession(configs[name].KinesisPort)

	if err != nil {
		logErr(err, "could not create kinesis client: %s")
	}

	kinesisClients[name] = kinesis.New(sess)

	return kinesisClients[name]
}

func runLocalstackContainer(name string, config configInput) {
	wait.Add(1)
	go doRunLocalstack(name, config)
}

func doRunLocalstack(name string, configMap configInput) {
	defer wait.Done()
	defer log.Printf("%s component of type %s is ready", name, "localstack")

	localConfig := &localstackConfig{}
	unmarshalConfig(configMap, localConfig)
	configs[name] = localConfig

	runContainer("gosoline_test_localstack", ContainerConfig{
		Repository: "localstack/localstack",
		Tag:        "0.10.3",
		Env: []string{
			"SERVICES=" + localConfig.Services,
		},
		PortBindings: PortBinding{
			"4575/tcp": fmt.Sprint(localConfig.SnsPort),
			"4576/tcp": fmt.Sprint(localConfig.SqsPort),
			"4568/tcp": fmt.Sprint(localConfig.KinesisPort),
			"4582/tcp": fmt.Sprint(localConfig.CloudwatchPort),
		},
		HealthCheck: func() error {
			err := &multierror.Error{}

			if localConfig.CloudwatchPort > 0 {
				cloudwatchClient := ProvideCloudwatchClient(name)
				_, errCloudwatch := cloudwatchClient.ListDashboards(&cloudwatch.ListDashboardsInput{})
				err = multierror.Append(errCloudwatch)
			}

			if localConfig.KinesisPort > 0 {
				kinesisClient := ProvideKinesisClient(name)
				_, errKinesis := kinesisClient.ListStreams(&kinesis.ListStreamsInput{})
				err = multierror.Append(errKinesis)
			}

			if localConfig.SqsPort > 0 {
				sqsClient := ProvideSqsClient(name)
				_, errSqs := sqsClient.ListQueues(&sqs.ListQueuesInput{})
				err = multierror.Append(errSqs)
			}

			if localConfig.SnsPort > 0 {
				snsClient := ProvideSnsClient(name)
				_, errSns := snsClient.ListTopics(&sns.ListTopicsInput{})
				err = multierror.Append(errSns)
			}

			return err.ErrorOrNil()
		},
	})
}
