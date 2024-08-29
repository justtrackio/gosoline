package env

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
)

const (
	DefaultAccessKeyID     = "gosoline"
	DefaultSecretAccessKey = "gosoline"
	DefaultToken           = ""
)

func GetDefaultStaticCredentials() credentials.StaticCredentialsProvider {
	return credentials.NewStaticCredentialsProvider(DefaultAccessKeyID, DefaultSecretAccessKey, DefaultToken)
}

func GetDefaultAwsSdkConfig() (aws.Config, error) {
	cfgOptions := []func(options *config.LoadOptions) error{
		config.WithRegion("eu-central-1"),
		config.WithCredentialsProvider(GetDefaultStaticCredentials()),
	}

	var err error
	var cfg aws.Config

	if cfg, err = config.LoadDefaultConfig(context.Background(), cfgOptions...); err != nil {
		return aws.Config{}, fmt.Errorf("unable to load aws sdk config: %w", err)
	}

	return cfg, nil
}
