package test

import (
	"fmt"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"log"
)

var dynamoDbConfigs map[string]*dynamoDbConfig
var dynamoDbClients = simpleCache{}

type dynamoDbConfig struct {
	Debug bool   `mapstructure:"debug"`
	Host  string `mapstructure:"host"`
	Port  int    `mapstructure:"port"`
}

type DynamoDbFixtures struct {
	Table *dynamodb.CreateTableInput
	Items []map[string]interface{}
}

func init() {
	dynamoDbClients = simpleCache{}
	dynamoDbConfigs = map[string]*dynamoDbConfig{}
}

func ProvideDynamoDbClient(name string) *dynamodb.DynamoDB {
	return dynamoDbClients.New(name, func() interface{} {
		sess, err := getSession(dynamoDbConfigs[name].Host, dynamoDbConfigs[name].Port)

		if err != nil {
			logErr(err, "could not create dynamodb client: %s")
		}

		return dynamodb.New(sess)
	}).(*dynamodb.DynamoDB)
}

func runDynamoDb(name string, config configInput) {
	wait.Add(1)
	go doRunDynamoDb(name, config)
}

func doRunDynamoDb(name string, configMap configInput) {
	defer wait.Done()
	defer log.Printf("%s component of type %s is ready", name, "dynamodb")

	config := &dynamoDbConfig{}
	unmarshalConfig(configMap, config)
	dynamoDbConfigs[name] = config

	runDynamoDbContainer(name, config.Debug)
}

func runDynamoDbContainer(name string, debug bool) {
	client := ProvideDynamoDbClient(name)

	containerName := fmt.Sprintf("gosoline_test_dynamodb_%s", name)

	runContainer(containerName, ContainerConfig{
		Repository: "amazon/dynamodb-local",
		Tag:        "latest",
		PortBindings: PortBinding{
			"8000/tcp": fmt.Sprint(dynamoDbConfigs[name].Port),
		},
		HealthCheck: func() error {
			_, err := client.ListTables(&dynamodb.ListTablesInput{})

			return err
		},
		PrintLogs: debug,
	})
}
