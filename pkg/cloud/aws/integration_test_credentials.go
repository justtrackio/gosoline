// +build integration

package aws

import "github.com/aws/aws-sdk-go/aws/credentials"

// see explanation in default_credentials.go
func GetDefaultCredentials() *credentials.Credentials {
	return credentials.NewChainCredentials([]credentials.Provider{
		&credentials.EnvProvider{},
		&credentials.SharedCredentialsProvider{Filename: "", Profile: ""},
		&credentials.StaticProvider{Value: credentials.Value{
			AccessKeyID:     "id",
			SecretAccessKey: "secret",
			SessionToken:    "token",
		}},
	})
}
