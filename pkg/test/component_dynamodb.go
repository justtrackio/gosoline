package test

import (
	"encoding/json"
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

type DynamoDbFixtures struct {
	Table *dynamodb.CreateTableInput
	Items []map[string]interface{}
}

func runDynamoDb(name string, config configMap) {
	wait.Add(1)
	go doRunDynamoDb(name, config)
}

func doRunDynamoDb(name string, config configMap) {
	defer wait.Done()
	defer log.Printf("%s component of type %s is ready", name, "dynamodb")

	fixturePath := configString(config, name, "fixtures")

	runDynamoDbContainer()
	jsonStr, err := ioutil.ReadFile(fixturePath)

	if err != nil {
		logErr(err, "could not read dynamodb fixtures")
	}

	fixtures := make(map[string]*DynamoDbFixtures)
	err = json.Unmarshal(jsonStr, &fixtures)

	if err != nil {
		logErr(err, "could not unmarshal dynamodb fixtures")
	}

	for name, fixture := range fixtures {
		createDynamoDbTable(name, fixture)
	}
}

func runDynamoDbContainer() {
	client := getDynamoDbClient()

	runContainer(dynamoDbContainerName, ContainerConfig{
		Repository: "amazon/dynamodb-local",
		Tag:        "latest",
		PortBindings: PortBinding{
			"8000/tcp": "4569",
		},
		HealthCheck: func() error {
			_, err := client.ListTables(&dynamodb.ListTablesInput{})
			return err
		},
	})
}

func createDynamoDbTable(table string, fixtures *DynamoDbFixtures) {
	input := fixtures.Table
	input.TableName = aws.String(table)
	input.ProvisionedThroughput = &dynamodb.ProvisionedThroughput{
		ReadCapacityUnits:  aws.Int64(1),
		WriteCapacityUnits: aws.Int64(1),
	}

	client := getDynamoDbClient()
	_, err := client.CreateTable(input)

	if err != nil {
		logErr(err, "could not create dynamodb table")
	}

	for _, item := range fixtures.Items {
		putDynamoDbItem(table, item)
	}
}

func putDynamoDbItem(table string, item map[string]interface{}) {
	attributes, err := dynamodbattribute.MarshalMap(item)

	if err != nil {
		logErr(err, "could not marshal dynamodb attributes")
	}

	input := &dynamodb.PutItemInput{
		TableName: aws.String(table),
		Item:      attributes,
	}

	client := getDynamoDbClient()
	_, err = client.PutItem(input)

	if err != nil {
		logErr(err, "could not put dynamodb item")
	}
}

func getDynamoDbClient() *dynamodb.DynamoDB {
	if dynamoDbClient != nil {
		return dynamoDbClient
	}

	config := &aws.Config{
		Region:   aws.String(endpoints.EuCentral1RegionID),
		Endpoint: aws.String("http://localhost:4569"),
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
