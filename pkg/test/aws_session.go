package test

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/endpoints"
	"github.com/aws/aws-sdk-go/aws/session"
	"net/http"
	"time"
)

var awsSessions simpleCache
var awsHttpClient = &http.Client{
	Timeout: time.Minute,
}

func getAwsSession(host string, port int) *session.Session {
	endpoint := fmt.Sprintf("http://%s:%d", host, port)

	s := awsSessions.New(endpoint, func() interface{} {
		return createNewSession(endpoint)
	})

	return s.(*session.Session)
}

func createNewSession(endpoint string) interface{} {
	config := &aws.Config{
		CredentialsChainVerboseErrors: aws.Bool(true),
		MaxRetries:                    aws.Int(30),
		Region:                        aws.String(endpoints.EuCentral1RegionID),
		Endpoint:                      aws.String(endpoint),
		HTTPClient:                    awsHttpClient,
		S3ForcePathStyle:              aws.Bool(true),
	}

	newSession, err := session.NewSession(config)

	if err != nil {
		panic(err)
	}

	return newSession
}
