package test

import (
	"fmt"
	"github.com/aws/aws-sdk-go/service/sns"
	"github.com/aws/aws-sdk-go/service/sqs"
	"log"
	"net"
	"strings"
	"time"
)

type snsSqsConfig struct {
	Debug   bool   `mapstructure:"debug"`
	Host    string `mapstructure:"host"`
	SnsPort int    `mapstructure:"sns_port"`
	SqsPort int    `mapstructure:"sqs_port"`
}

var snsSqsConfigs map[string]*snsSqsConfig

var localstackClients = simpleCache{}

func init() {
	snsSqsConfigs = map[string]*snsSqsConfig{}
	localstackClients = simpleCache{}
}

func onDestroy() {
	snsSqsConfigs = map[string]*snsSqsConfig{}
	localstackClients = simpleCache{}
}

func ProvideSnsClient(name string) *sns.SNS {
	return localstackClients.New("sns-"+name, func() interface{} {
		sess, err := getSession(snsSqsConfigs[name].Host, snsSqsConfigs[name].SnsPort)

		if err != nil {
			logErr(err, "could not create sns client: %s")
		}

		return sns.New(sess)
	}).(*sns.SNS)
}

func ProvideSqsClient(name string) *sqs.SQS {
	return localstackClients.New("sqs-"+name, func() interface{} {
		sess, err := getSession(snsSqsConfigs[name].Host, snsSqsConfigs[name].SqsPort)

		if err != nil {
			logErr(err, "could not create sqs client: %s")
		}

		return sqs.New(sess)
	}).(*sqs.SQS)
}

func runSnsSqs(name string, config configInput) {
	wait.Add(1)
	go doRunSnsSqs(name, config)
}

func doRunSnsSqs(name string, configMap configInput) {
	defer wait.Done()
	defer log.Printf("%s component of type %s is ready", name, "sns_sqs")

	localConfig := &snsSqsConfig{}
	unmarshalConfig(configMap, localConfig)
	snsSqsConfigs[name] = localConfig

	services := "SERVICES=" + strings.Join([]string{
		"sns",
		"sqs",
	}, ",")

	env := []string{services}

	if localConfig.Debug {
		env = append(env, "DEBUG=1")
	}

	containerName := fmt.Sprintf("gosoline_test_sns_sqs_%s", name)

	runContainer(containerName, ContainerConfig{
		Repository: "localstack/localstack",
		Tag:        "0.10.7",
		Env:        env,
		PortBindings: PortBinding{
			"4575/tcp": fmt.Sprint(localConfig.SnsPort),
			"4576/tcp": fmt.Sprint(localConfig.SqsPort),
		},
		HealthCheck: localstackHealthCheck(containerName),
		OnDestroy:   onDestroy,
		PrintLogs:   localConfig.Debug,
	})

	c, err := dockerPool.Client.InspectContainer(containerName)

	if err != nil {
		logErr(err, "could not inspect container")
	}

	address := c.NetworkSettings.Networks["bridge"].IPAddress

	if isReachable(address + ":4575") {
		fmt.Println("overriding host", address)
		snsSqsConfigs[name].Host = address
	}
}

func isReachable(address string) bool {
	timeout := time.Duration(5) * time.Second
	conn, err := net.DialTimeout("tcp", address, timeout)
	defer func() {
		err := conn.Close()

		if err != nil {
			logErr(err, "failed to close connection")
		}
	}()

	if err != nil {
		fmt.Println(err)
		return false
	}

	fmt.Println("connection established between localhost and", address)
	fmt.Println("remote address", conn.RemoteAddr().String())
	fmt.Println("local address", conn.LocalAddr().String())

	return true
}
