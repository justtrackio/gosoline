package test

import (
	"encoding/json"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"io/ioutil"
	"log"
	"sync"
)

const dynamoDbContainerName = "gosoline_test_dynamoDb"

var dynamoDbConfigs map[string]*dynamoDbConfig
var dynamoDbClients map[string]*dynamodb.DynamoDB
var dynamoDbLck sync.Mutex

type dynamoDbConfig struct {
	Fixtures string `mapstructure:"fixtures"`
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
}

type DynamoDbFixtures struct {
	Table *dynamodb.CreateTableInput
	Items []map[string]interface{}
}

func init() {
	dynamoDbConfigs = map[string]*dynamoDbConfig{}
	dynamoDbClients = map[string]*dynamodb.DynamoDB{}
}

func ProvideDynamoDbClient(name string) *dynamodb.DynamoDB {
	dynamoDbLck.Lock()
	defer dynamoDbLck.Unlock()

	_, ok := dynamoDbClients[name]
	if ok {
		return dynamoDbClients[name]
	}

	sess, err := getSession(dynamoDbConfigs[name].Host, dynamoDbConfigs[name].Port)

	if err != nil {
		logErr(err, "could not create dynamodb client: %s")
	}

	dynamoDbClients[name] = dynamodb.New(sess)

	return dynamoDbClients[name]
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

	runDynamoDbContainer(name)
	jsonStr, err := ioutil.ReadFile(config.Fixtures)

	if err != nil {
		logErr(err, "could not read dynamodb fixtures")
	}

	fixtures := make(map[string]*DynamoDbFixtures)
	err = json.Unmarshal(jsonStr, &fixtures)

	if err != nil {
		logErr(err, "could not unmarshal dynamodb fixtures")
	}

	for tablename, fixture := range fixtures {
		createDynamoDbTable(name, tablename, fixture)
	}
}

func runDynamoDbContainer(name string) {
	client := ProvideDynamoDbClient(name)

	runContainer(dynamoDbContainerName, ContainerConfig{
		Repository: "amazon/dynamodb-local",
		Tag:        "latest",
		PortBindings: PortBinding{
			"8000/tcp": fmt.Sprint(dynamoDbConfigs[name].Port),
		},
		HealthCheck: func() error {
			_, err := client.ListTables(&dynamodb.ListTablesInput{})

			return err
		},
	})
}

func createDynamoDbTable(name string, table string, fixtures *DynamoDbFixtures) {
	input := fixtures.Table
	input.TableName = aws.String(table)
	input.ProvisionedThroughput = &dynamodb.ProvisionedThroughput{
		ReadCapacityUnits:  aws.Int64(1),
		WriteCapacityUnits: aws.Int64(1),
	}

	client := ProvideDynamoDbClient(name)
	_, err := client.CreateTable(input)

	if err != nil {
		logErr(err, "could not create dynamodb table")
	}

	for _, item := range fixtures.Items {
		putDynamoDbItem(name, table, item)
	}
}

func putDynamoDbItem(name string, table string, item map[string]interface{}) {
	attributes, err := dynamodbattribute.MarshalMap(item)

	if err != nil {
		logErr(err, "could not marshal dynamodb attributes")
	}

	input := &dynamodb.PutItemInput{
		TableName: aws.String(table),
		Item:      attributes,
	}

	client := ProvideDynamoDbClient(name)
	_, err = client.PutItem(input)

	if err != nil {
		logErr(err, "could not put dynamodb item")
	}
}
