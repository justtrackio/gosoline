package aws

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsCfg "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/credentials/stscreds"
	"github.com/aws/aws-sdk-go-v2/service/sts"
)

func GetCredentialsProvider(ctx context.Context, settings ClientSettings) (aws.CredentialsProvider, error) {
	if settings.Credentials.AccessKeyID != "" {
		return credentials.NewStaticCredentialsProvider(settings.Credentials.AccessKeyID, settings.Credentials.SecretAccessKey, settings.Credentials.SessionToken), nil
	}

	if len(settings.AssumeRole) > 0 {
		return GetAssumeRoleCredentialsProvider(ctx, settings.AssumeRole)
	}

	return GetDefaultProvider(), nil
}

func GetAssumeRoleCredentialsProvider(ctx context.Context, roleArn string) (aws.CredentialsProvider, error) {
	var err error
	var conf aws.Config

	if conf, err = awsCfg.LoadDefaultConfig(ctx); err != nil {
		return nil, fmt.Errorf("can not load default aws config: %w", err)
	}

	stsClient := sts.NewFromConfig(conf)

	return stscreds.NewAssumeRoleProvider(stsClient, roleArn), nil
}
