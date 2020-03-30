package test

import (
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/mon"
	"github.com/aws/aws-sdk-go/service/dynamodb"
)

type dynamoDbSettings struct {
	*mockSettings
	Port int `cfg:"port" default:"0"`
}

type dynamoDbComponent struct {
	mockComponentBase
	settings *dynamoDbSettings
	clients  *simpleCache
}

func (d *dynamoDbComponent) Boot(config cfg.Config, _ mon.Logger, runner *dockerRunner, settings *mockSettings, name string) {
	d.name = name
	d.runner = runner
	d.settings = &dynamoDbSettings{
		mockSettings: settings,
	}
	d.clients = &simpleCache{}
	key := fmt.Sprintf("mocks.%s", name)
	config.UnmarshalKey(key, d.settings)
}

func (d *dynamoDbComponent) Start() error {
	containerName := fmt.Sprintf("gosoline_test_dynamodb_%s", d.name)

	return d.runner.Run(containerName, containerConfig{
		Repository: "amazon/dynamodb-local",
		Tag:        "latest",
		PortBindings: portBinding{
			"8000/tcp": fmt.Sprint(d.settings.Port),
		},
		PortMappings: portMapping{
			"8000/tcp": &d.settings.Port,
		},
		HealthCheck: func() error {
			client := d.provideDynamoDbClient()

			_, err := client.ListTables(&dynamodb.ListTablesInput{})

			return err
		},
		PrintLogs:   d.settings.Debug,
		ExpireAfter: d.settings.ExpireAfter,
	})
}

func (d *dynamoDbComponent) provideDynamoDbClient() *dynamodb.DynamoDB {
	return d.clients.New(d.name, func() interface{} {
		sess := getAwsSession(d.settings.Host, d.settings.Port)

		return dynamodb.New(sess)
	}).(*dynamodb.DynamoDB)
}
