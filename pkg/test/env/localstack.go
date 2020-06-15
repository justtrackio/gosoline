package env

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

func localstackHealthCheck(service string) func(container *container) error {
	return func(container *container) error {
		binding := container.bindings["8080/tcp"]
		url := fmt.Sprintf("http://%s:%s/health?reload", binding.host, binding.port)

		var err error
		var resp *http.Response
		var body []byte
		var status = make(map[string]map[string]string)

		if resp, err = http.Get(url); err != nil {
			return err
		}

		if body, err = ioutil.ReadAll(resp.Body); err != nil {
			return err
		}

		if err = json.Unmarshal(body, &status); err != nil {
			return err
		}

		if _, ok := status["services"]; !ok {
			return fmt.Errorf("sns service is not up yet")
		}

		if _, ok := status["services"][service]; !ok {
			return fmt.Errorf("%s service is not up yet", service)
		}

		if status["services"][service] != "running" {
			return fmt.Errorf("%s service is in %s state", service, status["services"]["sns"])
		}

		return nil
	}
}
