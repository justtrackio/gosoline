package test

import (
	"log"
)

func runCloudwatchContainer(name string, config configInput) {
	wait.Add(1)
	go doRunCloudwatch(name, config)
}

func doRunCloudwatch(name string, config configInput) {
	defer wait.Done()
	defer log.Printf("%s component of type %s is ready", name, "cloudwatch")

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
			//_, err := client.ListTables(&dynamodb.ListTablesInput{})
			return err
		},
	})
}
