//go:build !integration
// +build !integration

package aws

import (
	"github.com/aws/aws-sdk-go/aws/credentials"
)

const (
	DefaultAccessKeyID     = ""
	DefaultSecretAccessKey = ""
	DefaultToken           = ""
)

// GetDefaultCredentials provides you with credentials to use. In an integration test, you will get the credentials
// matching your environment or some static credentials if there are no credentials in your environment. Outside of
// tests, you get this implementation which tells the AWS SDK to use the default credentials (as if you didn't specify
// any credentials at all).
func GetDefaultCredentials() *credentials.Credentials {
	return nil
}
