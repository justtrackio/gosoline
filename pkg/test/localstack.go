package test

import (
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/justtrackio/gosoline/pkg/encoding/json"
)

type localstackHealthcheck struct {
	Services localstackHealthcheckServices
}

type localstackHealthcheckServices struct {
	Cloudwatch string
	Kinesis    string
	S3         string
	SNS        string
	SQS        string
}

func localstackHealthCheck(settings *mockSettings, services ...string) func() error {
	return func() error {
		url := fmt.Sprintf("http://%s:%d/health", settings.Host, settings.Port)
		resp, err := http.Get(url)
		if err != nil {
			return err
		}

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return err
		}

		localstackHealthcheck := &localstackHealthcheck{}
		err = json.Unmarshal(body, localstackHealthcheck)

		if err != nil {
			return err
		}

		for _, service := range services {
			switch service {
			case "cloudwatch":
				if localstackHealthcheck.Services.Cloudwatch != "running" {
					return fmt.Errorf("service cloudwatch is in state %s", localstackHealthcheck.Services.Cloudwatch)
				}
			case componentKinesis:
				if localstackHealthcheck.Services.Kinesis != "running" {
					return fmt.Errorf("service kinesis is in state %s", localstackHealthcheck.Services.Kinesis)
				}
			case componentS3:
				if localstackHealthcheck.Services.S3 != "running" {
					return fmt.Errorf("service s3 is in state %s", localstackHealthcheck.Services.S3)
				}
			case componentSns:
				if localstackHealthcheck.Services.SNS != "running" {
					return fmt.Errorf("service sns is in state %s", localstackHealthcheck.Services.SNS)
				}
			case componentSqs:
				if localstackHealthcheck.Services.SQS != "running" {
					return fmt.Errorf("service sqs is in state %s", localstackHealthcheck.Services.SQS)
				}
			}
		}

		return nil
	}
}
