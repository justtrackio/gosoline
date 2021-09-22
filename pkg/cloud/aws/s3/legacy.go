package s3

import (
	"net/http"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/justtrackio/gosoline/pkg/cfg"
	gosoAws "github.com/justtrackio/gosoline/pkg/cloud/aws"
)

func GetLegacyConfig(config cfg.Config, name string, optFns ...ClientOption) *aws.Config {
	clientCfg := getClientConfig(config, name, optFns...)
	awsCfg := &aws.Config{
		CredentialsChainVerboseErrors: aws.Bool(true),
		Endpoint:                      aws.String(clientCfg.Settings.Endpoint),
		Region:                        aws.String(clientCfg.Settings.Region),
		HTTPClient: &http.Client{
			Timeout: 1 * time.Minute,
		},
		MaxRetries:       aws.Int(30),
		S3ForcePathStyle: aws.Bool(clientCfg.Settings.UsePathStyle),
	}

	creds := gosoAws.UnmarshalCredentials(config)
	if creds != nil {
		awsCfg.Credentials = credentials.NewStaticCredentials(creds.AccessKeyID, creds.SecretAccessKey, creds.SessionToken)
	}

	return awsCfg
}
