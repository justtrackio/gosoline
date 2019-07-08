package test

import (
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"log"
)

func runCloudwatchContainer(name string, config configMap) {
	wait.Add(1)
	go doRunCloudwatch(name, config)
}

func doRunCloudwatch(name string, config configMap) {
	defer wait.Done()
	defer log.Printf("%s component of type %s is ready", name, "cloudwatch")

	client := getDynamoDbClient()

	runContainer("gosoline_test_cloudwatch", ContainerConfig{
		Repository: "localstack/localstack",
		Tag:        "latest",
		Env: []string{
			"SERVICES=cloudwatch",
		},
		PortBindings: PortBinding{
			"4582/tcp": "4582",
		},
		HealthCheck: func() error {
			_, err := client.ListTables(&dynamodb.ListTablesInput{})
			return err
		},
	})
}
