package test

import (
	"fmt"
	"github.com/applike/gosoline/pkg/mdl"
	"github.com/ory/dockertest"
	"github.com/ory/dockertest/docker"
	"log"
	"sync"
)

var err error
var wait sync.WaitGroup
var dockerPool *dockertest.Pool
var dockerNetwork *docker.Network
var dockerResources []*dockertest.Resource
var cfgFilename = "config.test.yml"

func init() {
	dockerPool, err = dockertest.NewPool("")
	dockerResources = make([]*dockertest.Resource, 0)

	if err != nil {
		log.Fatalf("Could not connect to docker: %s", err)
	}

	dockerNetwork, err = dockerPool.Client.CreateNetwork(docker.CreateNetworkOptions{
		Name: "gosoline_test",
	})

	if err != nil {
		log.Fatalf("Could not connect to docker: %s", err)
	}
}

func logErr(err error, msg string) {
	Shutdown()
	log.Println(msg)
	log.Fatal(err)
}

func Boot(configFilename *string) {
	if len(mdl.EmptyStringIfNil(configFilename)) > 0 {
		cfgFilename = *configFilename
	}

	config := readConfig(cfgFilename)

	for name, mockConfig := range config.Mocks {
		bootComponent(name, mockConfig)
	}

	wait.Wait()

	log.Println("test environment up and running")
	fmt.Println()
}

func bootComponent(name string, mockConfig configInput) {
	component := mockConfig["component"]

	switch component {
	case "cloudwatch":
		runCloudwatch(name, mockConfig)
	case "dynamodb":
		runDynamoDb(name, mockConfig)
	case "elasticsearch":
		runElasticsearch(name, mockConfig)
	case "kinesis":
		runKinesis(name, mockConfig)
	case "mysql":
		runMysql(name, mockConfig)
	case "redis":
		runRedis(name, mockConfig)
	case "sns":
		runSns(name, mockConfig)
	case "sqs":
		runSqs(name, mockConfig)
	case "wiremock":
		runWiremock(name, mockConfig)
	default:
		err := fmt.Errorf("unknown component '%s'", component)
		logErr(err, err.Error())
	}
}

func Shutdown() {
	for _, res := range dockerResources {
		if err := dockerPool.Purge(res); err != nil {
			log.Fatalf("Could not purge resource: %s", err)
		}
	}

	if err := dockerPool.Client.RemoveNetwork(dockerNetwork.ID); err != nil {
		log.Printf("Could not remove test network (%s): %s ", dockerNetwork.ID, err.Error())
	}
}
