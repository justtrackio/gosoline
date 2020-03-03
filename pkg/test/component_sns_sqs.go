package test

import (
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/aws/aws-sdk-go/service/sns"
	"github.com/aws/aws-sdk-go/service/sqs"
	"log"
	"net"
	"strings"
	"time"
)

type snsSqsSettings struct {
	*mockSettings
	SnsPort int `cfg:"sns_port"`
	SqsPort int `cfg:"sqs_port"`
}

type snsSqsComponent struct {
	name     string
	settings *snsSqsSettings
	clients  *simpleCache
}

func (c *snsSqsComponent) Boot(name string, config cfg.Config, settings *mockSettings) {
	c.name = name
	c.clients = &simpleCache{}
	c.settings = &snsSqsSettings{
		mockSettings: settings,
	}
	key := fmt.Sprintf("mocks.%s", name)
	config.UnmarshalKey(key, c.settings)
}

func (c *snsSqsComponent) Run(runner *dockerRunner) {
	defer log.Printf("%s component of type %s is ready", c.name, "sns_sqs")

	services := "SERVICES=" + strings.Join([]string{
		"sns",
		"sqs",
	}, ",")

	env := []string{services}

	if c.settings.Debug {
		env = append(env, "DEBUG=1")
	}

	containerName := fmt.Sprintf("gosoline_test_sns_sqs_%s", c.name)

	runner.Run(containerName, containerConfig{
		Repository: "localstack/localstack",
		Tag:        "0.10.7",
		Env:        env,
		PortBindings: portBinding{
			"4575/tcp": fmt.Sprint(c.settings.SnsPort),
			"4576/tcp": fmt.Sprint(c.settings.SqsPort),
		},
		HealthCheck: localstackHealthCheck(runner, containerName),
		PrintLogs:   c.settings.Debug,
	})

	address := runner.GetIpAddress(containerName)

	if isReachable(address + ":4575") {
		log.Println("overriding host", address)
		c.settings.Host = address
	}
}

func (c *snsSqsComponent) ProvideClient(clientType string) interface{} {
	switch clientType {
	case "sns":
		return c.provideSnsClient()
	case "sqs":
		return c.provideSqsClient()
	}
	panic("unknown clientType " + clientType)
}

func (c *snsSqsComponent) provideSnsClient() *sns.SNS {
	return c.clients.New(fmt.Sprintf("sns-%s", c.name), func() interface{} {
		sess, err := getAwsSession(c.settings.Host, c.settings.SnsPort)

		if err != nil {
			panic(fmt.Errorf("could not create sns client: %s : %w", c.name, err))
		}

		return sns.New(sess)
	}).(*sns.SNS)
}

func (c *snsSqsComponent) provideSqsClient() *sqs.SQS {
	return c.clients.New(fmt.Sprintf("sqs-%s", c.name), func() interface{} {
		sess, err := getAwsSession(c.settings.Host, c.settings.SqsPort)

		if err != nil {
			panic(fmt.Errorf("could not create sqs client %s : %w", c.name, err))
		}

		return sqs.New(sess)
	}).(*sqs.SQS)
}

func isReachable(address string) bool {
	timeout := time.Duration(1) * time.Second
	conn, err := net.DialTimeout("tcp", address, timeout)

	if err != nil {
		return false
	}

	defer func() {
		err := conn.Close()

		if err != nil {
			panic(fmt.Errorf("failed to close connection : %w", err))
		}
	}()

	return true
}
