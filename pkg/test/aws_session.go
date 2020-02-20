package test

import (
	"fmt"
	"github.com/applike/gosoline/pkg/mdl"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/endpoints"
	"github.com/aws/aws-sdk-go/aws/session"
	"log"
)

func getSession(host string, port int) (*session.Session, error) {
	endpoint := fmt.Sprintf("http://%s:%d", host, port)
	log.Println("endpoint is: " + endpoint)

	config := &aws.Config{
		MaxRetries: mdl.Int(5),
		Region:     aws.String(endpoints.EuCentral1RegionID),
		Endpoint:   aws.String(endpoint),
	}

	return session.NewSession(config)
}
