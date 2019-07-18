package test

import (
	"encoding/json"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/endpoints"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"io/ioutil"
	"log"
	"net/http"
	"time"
)

const dynamoDbContainerName = "gosoline_test_dynamoDb"

var dynamoDbClient *dynamodb.DynamoDB

type dynamoDbConfig struct {
	Fixtures string `mapstructure:"fixtures"`
	Port     int    `mapstructure:"port"`
}

type DynamoDbFixtures struct {
	Table *dynamodb.CreateTableInput
	Items []map[string]interface{}
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

	runDynamoDbContainer(config.Port)
	jsonStr, err := ioutil.ReadFile(config.Fixtures)

	if err != nil {
		logErr(err, "could not read dynamodb fixtures")
	}

	fixtures := make(map[string]*DynamoDbFixtures)
	err = json.Unmarshal(jsonStr, &fixtures)

	if err != nil {
		logErr(err, "could not unmarshal dynamodb fixtures")
	}

	for name, fixture := range fixtures {
		createDynamoDbTable(config.Port, name, fixture)
	}
}

func runDynamoDbContainer(port int) {
	client := getDynamoDbClient(port)

	runContainer(dynamoDbContainerName, ContainerConfig{
		Repository: "amazon/dynamodb-local",
		Tag:        "latest",
		PortBindings: PortBinding{
			"8000/tcp": fmt.Sprint(port),
		},
		HealthCheck: func() error {
			_, err := client.ListTables(&dynamodb.ListTablesInput{})
			return err
		},
	})
}

func createDynamoDbTable(port int, table string, fixtures *DynamoDbFixtures) {
	input := fixtures.Table
	input.TableName = aws.String(table)
	input.ProvisionedThroughput = &dynamodb.ProvisionedThroughput{
		ReadCapacityUnits:  aws.Int64(1),
		WriteCapacityUnits: aws.Int64(1),
	}

	client := getDynamoDbClient(port)
	_, err := client.CreateTable(input)

	if err != nil {
		logErr(err, "could not create dynamodb table")
	}

	for _, item := range fixtures.Items {
		putDynamoDbItem(port, table, item)
	}
}

func putDynamoDbItem(port int, table string, item map[string]interface{}) {
	attributes, err := dynamodbattribute.MarshalMap(item)

	if err != nil {
		logErr(err, "could not marshal dynamodb attributes")
	}

	input := &dynamodb.PutItemInput{
		TableName: aws.String(table),
		Item:      attributes,
	}

	client := getDynamoDbClient(port)
	_, err = client.PutItem(input)

	if err != nil {
		logErr(err, "could not put dynamodb item")
	}
}

func getDynamoDbClient(port int) *dynamodb.DynamoDB {
	if dynamoDbClient != nil {
		return dynamoDbClient
	}

	host := fmt.Sprintf("http://localhost:%d", port)

	config := &aws.Config{
		Region:   aws.String(endpoints.EuCentral1RegionID),
		Endpoint: aws.String(host),
		HTTPClient: &http.Client{
			Timeout: 1 * time.Minute,
		},
	}

	sess, err := session.NewSession(config)

	if err != nil {
		logErr(err, "could not create dynamodb client: %s")
	}

	dynamoDbClient = dynamodb.New(sess)

	return dynamoDbClient
}
