package test

import (
	"fmt"
	"github.com/applike/gosoline/pkg/mdl"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/endpoints"
	"github.com/aws/aws-sdk-go/aws/session"
	"net/http"
	"time"
)

func getSession(host string, port int) (*session.Session, error) {
	endpoint := fmt.Sprintf("http://%s:%d", host, port)

	config := &aws.Config{
		MaxRetries: mdl.Int(5),
		Region:     aws.String(endpoints.EuCentral1RegionID),
		Endpoint:   aws.String(endpoint),
		HTTPClient: &http.Client{
			Timeout: 1 * time.Minute,
		},
	}

	return session.NewSession(config)
}
