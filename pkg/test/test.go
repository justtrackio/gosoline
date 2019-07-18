package test

import (
	"fmt"
	"github.com/ory/dockertest"
	"log"
	"sync"
)

var err error
var wait sync.WaitGroup
var dockerPool *dockertest.Pool
var dockerResources []*dockertest.Resource

func init() {
	dockerPool, err = dockertest.NewPool("")
	dockerResources = make([]*dockertest.Resource, 0)

	if err != nil {
		log.Fatalf("Could not connect to docker: %s", err)
	}
}

func logErr(err error, msg string) {
	Shutdown()
	log.Println(msg)
	log.Fatal(err)
}

func Boot() {
	config := readConfig()

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
	case "dynamodb":
		runDynamoDb(name, mockConfig)
	case "wiremock":
		runWiremock(name, mockConfig)
	case "elasticsearch":
		runElasticsearch(name, mockConfig)
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
}
