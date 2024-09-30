//go:build !integration
// +build !integration

package aws

import (
	"github.com/aws/aws-sdk-go-v2/aws"
)

const (
	DefaultAccessKeyID     = ""
	DefaultSecretAccessKey = ""
	DefaultToken           = ""
)

func GetDefaultProvider() aws.CredentialsProvider {
	return nil
}
