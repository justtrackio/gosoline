//go:build integration
// +build integration

package aws

import (
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
)

const (
	DefaultAccessKeyID     = "gosoline"
	DefaultSecretAccessKey = "gosoline"
	DefaultToken           = "token"
)

func GetDefaultProvider() aws.CredentialsProvider {
	return credentials.NewStaticCredentialsProvider(DefaultAccessKeyID, DefaultSecretAccessKey, DefaultToken)
}
