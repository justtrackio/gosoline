package test

import (
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"log"
)

type dynamoDbSettings struct {
	*mockSettings
	Port int `cfg:"port"`
}

type dynamoDbComponent struct {
	name     string
	settings *dynamoDbSettings
	clients  *simpleCache
}

func (m *dynamoDbComponent) Boot(name string, config cfg.Config, settings *mockSettings) {
	m.name = name
	m.settings = &dynamoDbSettings{
		mockSettings: settings,
	}
	m.clients = &simpleCache{}
	key := fmt.Sprintf("mocks.%s", name)
	config.UnmarshalKey(key, m.settings)
}

func (m *dynamoDbComponent) Run(runner *dockerRunner) {
	defer log.Printf("%s component of type %s is ready", m.name, "dynamodb")

	containerName := fmt.Sprintf("gosoline_test_dynamodb_%s", m.name)

	runner.Run(containerName, containerConfig{
		Repository: "amazon/dynamodb-local",
		Tag:        "latest",
		PortBindings: portBinding{
			"8000/tcp": fmt.Sprint(m.settings.Port),
		},
		HealthCheck: func() error {
			client := m.provideDynamoDbClient()

			_, err := client.ListTables(&dynamodb.ListTablesInput{})

			return err
		},
		PrintLogs: m.settings.Debug,
	})
}

func (m *dynamoDbComponent) ProvideClient(string) interface{} {
	return m.provideDynamoDbClient()
}

func (m *dynamoDbComponent) provideDynamoDbClient() *dynamodb.DynamoDB {
	return m.clients.New(m.name, func() interface{} {
		sess, err := getAwsSession(m.settings.Host, m.settings.Port)

		if err != nil {
			panic(fmt.Errorf("could not create dynamodb client %s, %w", m.name, err))
		}

		return dynamodb.New(sess)
	}).(*dynamodb.DynamoDB)
}
