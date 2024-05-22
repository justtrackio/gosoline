//go:build integration
// +build integration

package aws

import (
	"github.com/aws/aws-sdk-go-v2/aws"
	credentialsv2 "github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go/aws/credentials"
)

const (
	DefaultAccessKeyID     = "gosoline"
	DefaultSecretAccessKey = "gosoline"
	DefaultToken           = "token"
)

// see explanation in default_credentials.go
func GetDefaultCredentials() *credentials.Credentials {
	return credentials.NewChainCredentials([]credentials.Provider{
		&credentials.EnvProvider{},
		&credentials.SharedCredentialsProvider{Filename: "", Profile: ""},
		&credentials.StaticProvider{Value: credentials.Value{
			AccessKeyID:     DefaultAccessKeyID,
			SecretAccessKey: DefaultSecretAccessKey,
			SessionToken:    DefaultToken,
		}},
	})
}

func GetDefaultProvider() aws.CredentialsProvider {
	return credentialsv2.NewStaticCredentialsProvider(DefaultAccessKeyID, DefaultSecretAccessKey, DefaultToken)
}
