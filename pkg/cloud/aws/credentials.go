package aws

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsCfg "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/credentials/stscreds"
	"github.com/aws/aws-sdk-go-v2/service/sts"
)

func GetCredentialsOption(ctx context.Context, settings ClientSettings) (func(options *awsCfg.LoadOptions) error, error) {
	if settings.Profile != "" {
		return awsCfg.WithSharedConfigProfile(settings.Profile), nil
	}

	var err error
	var credentialsProvider aws.CredentialsProvider

	if credentialsProvider, err = GetCredentialsProvider(ctx, settings); err != nil {
		return nil, fmt.Errorf("can not get credentials provider: %w", err)
	}

	if credentialsProvider == nil {
		return nil, nil
	}

	return awsCfg.WithCredentialsProvider(credentialsProvider), nil
}

func GetCredentialsProvider(ctx context.Context, settings ClientSettings) (aws.CredentialsProvider, error) {
	if settings.Credentials.AccessKeyID != "" {
		return credentials.NewStaticCredentialsProvider(settings.Credentials.AccessKeyID, settings.Credentials.SecretAccessKey, settings.Credentials.SessionToken), nil
	}

	if settings.AssumeRole != "" {
		return GetAssumeRoleCredentialsProvider(ctx, settings.AssumeRole)
	}

	if webIdentitySettings := GetWebIdentitySettings(); settings.UseWebIdentity && webIdentitySettings != nil {
		return GetWebIdentityRoleProvider(ctx, webIdentitySettings)
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

func GetWebIdentityRoleProvider(ctx context.Context, webIdentitySettings *WebIdentitySettings) (aws.CredentialsProvider, error) {
	var err error
	var conf aws.Config

	if conf, err = awsCfg.LoadDefaultConfig(ctx); err != nil {
		return nil, fmt.Errorf("can not load default aws config: %w", err)
	}
	client := sts.NewFromConfig(conf)
	credsCache := aws.NewCredentialsCache(stscreds.NewWebIdentityRoleProvider(
		client,
		webIdentitySettings.RoleARN,
		stscreds.IdentityTokenFile(webIdentitySettings.TokenFilePath),
		func(o *stscreds.WebIdentityRoleOptions) {
			o.RoleSessionName = fmt.Sprintf("gosoline-%s", time.Now().Format("20060102150405"))
		}))

	return credsCache, nil
}

func GetWebIdentitySettings() *WebIdentitySettings {
	awsRoleARN := os.Getenv("AWS_ROLE_ARN")
	awsWebIdentityTokenFile := os.Getenv("AWS_WEB_IDENTITY_TOKEN_FILE")
	if awsRoleARN == "" || awsWebIdentityTokenFile == "" {
		return nil
	}

	settings := &WebIdentitySettings{
		TokenFilePath: awsWebIdentityTokenFile,
		RoleARN:       awsRoleARN,
	}

	return settings
}
