package test

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
)

type wiremockConfig struct {
	Mocks string `mapstructure:"mocks"`
	Host  string `mapstructure:"host"`
	Port  int    `mapstructure:"port"`
}

func runWiremock(name string, config configInput) {
	wait.Add(1)
	go doRunWiremock(name, config)
}

func doRunWiremock(name string, configMap configInput) {
	defer wait.Done()
	defer log.Printf("%s component of type %s is ready", name, "wiremock")

	config := &wiremockConfig{}
	unmarshalConfig(configMap, config)
	url := fmt.Sprintf("http://%s:%d/__admin", config.Host, config.Port)

	runContainer("gosoline_test_wiremock", ContainerConfig{
		Repository: "rodolpheche/wiremock",
		Tag:        "latest",
		PortBindings: PortBinding{
			"8080/tcp": fmt.Sprint(config.Port),
		},
		HealthCheck: func() error {
			_, err := http.Get(url)

			return err
		},
	})

	jsonStr, err := ioutil.ReadFile(config.Mocks)

	if err != nil {
		logErr(err, "could not read http mock configuration")
	}

	_, err = http.Post(url+"/mappings/import", "application/json", bytes.NewBuffer(jsonStr))

	if err != nil {
		logErr(err, "could not send stubs to wiremock")
	}
}
